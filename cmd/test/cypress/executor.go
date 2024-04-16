package cypress

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
	errMissingRepo = errors.New("missing test suite repository")
)

// CypressTestExecutor implements TestExecutor interface for running k6 test suites
type CypressTestExecutor struct {
	Log *slog.Logger

	TargetDir          string
	TestSuiteRepo      string
	TestReportJsonPath string
	WorkingDir         string
}

// NewCypressTestExecutor creates a new instance of CypressTestExecutor
func NewCypressTestExecutor(
	log *slog.Logger,
	testSuiteRepo string,
	targetDir string,
	jsonReportPath string,
	workingDir string,
) *CypressTestExecutor {
	return &CypressTestExecutor{
		Log:                log,
		TestSuiteRepo:      testSuiteRepo,
		TestReportJsonPath: jsonReportPath,
		TargetDir:          targetDir,
		WorkingDir:         workingDir,
	}
}

func (t *CypressTestExecutor) Name() string {
	return "Cypress"
}

// go run . test test --test-type smoke --runner cypress --working-dir e2e --test-suite-repo git@github.com:grafana/plugins-private
// execute test suite
func (t *CypressTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.TestSuiteRepo == "" {
		return executor.SuiteRunSummary{}, errMissingRepo
	}

	testingDir := utils.GetTestingDirectory(t.TargetDir, t.TestSuiteRepo)
	workingDir := testingDir + t.WorkingDir

	err := utils.CloneRepo(testingDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to import repo: %s", err.Error())
	}

	err = t.prepareCodebase(workingDir)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(workingDir)
	if err != nil {
		// attempt to parse json report if there is a error
		t.Log.Info("Cypress processes exited with code 1", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s%s", workingDir, t.TestReportJsonPath))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing Cypress report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}

func (t *CypressTestExecutor) prepareCodebase(testingDir string) error {
	err := utils.CloneRepo(testingDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return fmt.Errorf("failed to import repo: %s", err.Error())
	}

	err = utils.ExecuteInDir(testingDir, func() error {
		installCmd := exec.Command("yarn", "install")
		if err := utils.ExecStdout(installCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to installing dependencies: %s", err.Error())
	}

	return nil
}

func (t *CypressTestExecutor) executeTests(testingDir string) error {
	err := utils.ExecuteInDir(testingDir, func() error {
		testRunCmd := exec.Command("yarn", "cypress", "run")
		if err := utils.ExecStdout(testRunCmd); err != nil {
			return err
		}

		return nil
	})

	return err
}
