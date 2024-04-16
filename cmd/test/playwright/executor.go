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

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log     *slog.Logger
	Verbose bool

	TargetDir          string
	TestSuiteRepo      string
	TestReportJsonPath string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	testSuiteRepo string,
	targetDir string,
	testReportJsonPath string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:                log,
		TestSuiteRepo:      testSuiteRepo,
		TargetDir:          targetDir,
		TestReportJsonPath: testReportJsonPath,
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
		return executor.SuiteRunSummary{}, errors.New("missing test suite repository")
	}

	testingDir := utils.GetTestingDirectory(t.TargetDir, t.TestSuiteRepo)

	err := t.prepareCodebase(testingDir)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(testingDir)
	if err != nil {
		// process might return exit code 1 if test fails but we still want to try to parse the report
		t.Log.Info("Playwright processes exited with code 1", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s%s", testingDir, t.TestReportJsonPath))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing playwright report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}

func (t *PlaywrightTestExecutor) prepareCodebase(testingDir string) error {
	err := utils.CloneRepo(testingDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return fmt.Errorf("failed to import repo: %s", err.Error())
	}

	err = utils.ExecuteInDir(testingDir, func() error {
		installCmd := exec.Command("yarn", "install")
		if err := utils.ExecStdout(installCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		installPlaywrightCmd := exec.Command("yarn", "playwright", "install")
		if err := utils.ExecStdout(installPlaywrightCmd); err != nil {
			return fmt.Errorf("installing playwright browsers: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to installing dependencies: %s", err.Error())
	}

	return nil
}

func (t *PlaywrightTestExecutor) executeTests(testingDir string) error {
	err := utils.ExecuteInDir(testingDir, func() error {
		testRunCmd := exec.Command("yarn", "playwright", "test")
		if err := utils.ExecStdout(testRunCmd); err != nil {
			return err
		}

		return nil
	})

	return err
}
