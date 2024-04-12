package playwright

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var (
	errMissingRepo           = errors.New("missing test suite repository")
	errMissingTargetDirError = errors.New("missing target directory to clone repository")
)

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log     *slog.Logger
	Verbose bool

	TargetDir         string
	TestSuiteRepo     string
	TestSuiteRevision string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	verbose bool,
	testSuiteRepo string,
	targetDir string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:           log,
		Verbose:       verbose,
		TestSuiteRepo: testSuiteRepo,
		TargetDir:     targetDir,
	}
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// go run . test test --test-suite /path/to/test/folder --test-type smoke --runner playwright --test-dir "./test-repo" --test-suite-repo git@github.com:grafana/grafana-plugin-tests
// execute test suite
func (t *PlaywrightTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {

	if t.TestSuiteRepo == "" {
		return executor.SuiteRunSummary{}, errMissingRepo
	}

	if t.TargetDir == "" {
		return executor.SuiteRunSummary{}, errMissingTargetDirError
	}

	err := ImportSetupRepo(t.TargetDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to import repo: %s", err.Error())
	}

	err = utils.ExecuteInDir(t.TargetDir, func() error {
		// idea: add a config in the repo with setup instructions
		installCmd := exec.Command("yarn", "test")
		if err := utils.ExecStdout(installCmd); err != nil {
			return err
		}

		return nil
	})

	// process might return exit code 1 but we still want to try to parse the report
	if err != nil {
		t.Log.Info("Playwright processes exited with code 1", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/playwright-report/report.json", t.TargetDir))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing playwright report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}
