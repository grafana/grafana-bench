package reporter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/hairyhenderson/go-codeowners"
)

var (
	ErrSendingNotification = errors.New("sending notification")
)
type NotificationOption func(r *notificationReporter)

// NotifyAll notifies all codeowners of the test results, not just owners of failed tests.
func NotifyPassing(notifyPassing bool) NotificationOption {
	return func(r *notificationReporter) {
		r.notifyPassing = notifyPassing
	}
}

type notificationReporter struct {
	notifier      notifier.Notifier
	notifyPassing bool
	codeowners    *codeowners.Codeowners 
}

// NewNotificationReporter returns a Reporter that notifies codeowners of test results using a Notifier
// Code owners are informed only of the tests they own.
// By default, only failed tests are notified. Use NotifyAll to notify all code owners.
// If the basedir is omitted (empty) the codeowners file will be searched in the current directory.
func NewNotificationReporter(
	baseDir string,
	notifier notifier.Notifier,
	opts ...NotificationOption,
) (SuiteRunReporter, error) {
	var err error

	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current working directory %w", err)
		}
	}

	c, err := codeowners.FromFileWithFS(os.DirFS(baseDir), ".")
	if err != nil && !errors.Is(err, codeowners.ErrNoCodeownersFound) {
		return nil, fmt.Errorf("finding codeowners %w", err)
	}

	reporter := &notificationReporter{
		notifier: notifier,
		codeowners: c,
	}

	for _, opt := range opts {
		opt(reporter)
	}

	return reporter, nil
}

func (r *notificationReporter) Report(
	ctx context.Context,
	suiteName string,
	suiteRevision string,
	runId string,
	suiteRunId string,
	suiteRun executor.SuiteRunSummary,
) error {
	// skip if codeowners wasn't found
	if r.codeowners == nil {
		return nil
	}
	// collects the test runs to be notified to each code owner
	recipients := map[string][]executor.TestRun{}

	for _, testRun := range suiteRun.TestRuns {
		owners := r.codeowners.Owners(filepath.Join(testRun.TestFolder, testRun.TestFile))
		for _, o := range owners {
			if r.notifyPassing || testRun.Status != executor.TestPassed {
				recipients[o] = append(recipients[o], testRun)
			}
		}
	}

	errs := []error{}
	for recipient, testRuns := range recipients {
		err := r.notifier.Notify(ctx, recipient, suiteRunId, testRuns)
		// it's ok not to have a notifications mapping for a codeowner
		if err != nil && !errors.Is(err, notifier.ErrNoMappingForCodeowner) {
			errs = append(errs, fmt.Errorf("recipient %q %w", recipient, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w %w", ErrSendingNotification, errors.Join(errs...))
	}

	return nil
}
