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

type NotificationOption func(r *notificationReporter)

// NotifyAll notifies all codeowners of the test results, not just owners of failed tests.
func NotifyAll(r *notificationReporter) {
	r.notifyAll = true
}

type notificationReporter struct {
	notifier  notifier.Notifier
	notifyAll bool
}

// NewNotificationReporter returns a Reporter that notifies codeowners of test results using a Notifier
// Code owners are informed only of the tests they own.
// By default, only failed tests are notified. Use NotifyAll to notify all code owners.
func NewNotificationReporter(notifier notifier.Notifier, opts ...NotificationOption) SuiteRunReporter {
	reporter := &notificationReporter{
		notifier: notifier,
	}

	for _, opt := range opts {
		opt(reporter)
	}

	return reporter
}

func (r *notificationReporter) Report(
	ctx context.Context,
	runId string,
	suiteRunId string,
	suite executor.TestSuite,
	suiteRun executor.SuiteRunSummary,
) error {
	// collects the test runs to be notified to each code owner
	recipients := map[string][]executor.TestRun{}

	c, err := codeowners.FromFileWithFS(os.DirFS(suite.BaseDir), suite.Path) 

	if errors.Is(err, codeowners.ErrNoCodeownersFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("finding codeowners %w", err)
	}

	for _, testRun := range suiteRun.TestRuns {
		owners := c.Owners(filepath.Join(testRun.TestFolder, testRun.TestFile))
		for _, o := range owners {
			if r.notifyAll || testRun.Status != executor.TestPassed {
				recipients[o] = append(recipients[o], testRun)
			}
		}
	}

	for recipient, testRuns := range recipients {
		err := r.notifier.Notify(ctx, recipient, suiteRunId, testRuns)
		if err != nil {
			return fmt.Errorf("sending notification to %q %w", recipient, err)
		}
	}

	return nil
}
