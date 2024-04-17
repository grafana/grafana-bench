package cypress

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/cmd/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var jsonOutputName = "test-results/cypress-report.json"

// CypressTestExecutor implements TestExecutor interface for running k6 test suites
type CypressTestExecutor struct {
	Log *slog.Logger

	TestSuiteRepo string
	PrepareCmd    string
	ExecuteCmd    string
	WorkingDir    string
}

// NewCypressTestExecutor creates a new instance of CypressTestExecutor
func NewCypressTestExecutor(
	log *slog.Logger,
	testSuiteRepo string,
	prepareCmd string,
	executeCmd string,
	workingDir string,
) *CypressTestExecutor {
	return &CypressTestExecutor{
		Log:           log,
		TestSuiteRepo: testSuiteRepo,
		PrepareCmd:    prepareCmd,
		ExecuteCmd:    executeCmd,
		WorkingDir:    workingDir,
	}
}

func (t *CypressTestExecutor) Name() string {
	return "Cypress"
}

// go run . test test --test-type smoke --test-suite /hi/ --runner cypress --br-working-dir /e2e --br-prepare-cmd "yarn install "--br-execute-cmd "yarn e2e:mock" --br-repo git@github.com:grafana/plugins-private --grafana-username e2e --grafana-password e2e
// execute test suite
func (t *CypressTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.TestSuiteRepo == "" {
		return executor.SuiteRunSummary{}, errors.New("missing test suite repository")
	}

	testingDir := utils.GetTestingDirectory(compile.TargetCloneDir, t.TestSuiteRepo)
	workingDir := testingDir + t.WorkingDir

	tc := compile.NewTestCompiler(t.Log, testingDir, t.TestSuiteRepo, "")
	err := tc.CloneRepo(context.TODO())
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to cloning codebase: %s", err.Error())
	}

	err = t.prepareCodebase(workingDir, t.PrepareCmd)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(workingDir, t.ExecuteCmd, suite.Path)
	if err != nil {
		// process might return exit code 1 if test fails but we still want to try to parse the report
		t.Log.Info("Playwright processes exited with code 1", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/%s", workingDir, jsonOutputName))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing Cypress report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}

func (t *CypressTestExecutor) prepareCodebase(testingDir string, prepareCmd string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		prepareCmd := exec.Command("bash", "-c", prepareCmd)
		if err := utils.ExecStdout(prepareCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		return nil
	})
}

func (t *CypressTestExecutor) executeTests(testingDir string, executeCmd string, testSuite string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		testRunCmd := exec.Command("bash", "-c", executeCmd)
		// testRunCmd := exec.Command("yarn", "cypress", "run")
		if err := utils.ExecStdout(testRunCmd); err != nil {
			return err
		}

		return nil
	})
}
