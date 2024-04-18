package cypress

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var jsonOutputName = "test-results/cypress-report.json"

// CypressTestExecutor implements TestExecutor interface for running k6 test suites
type CypressTestExecutor struct {
	Log *slog.Logger

	TargetDir  string
	PrepareCmd string
	ExecuteCmd string
}

// NewCypressTestExecutor creates a new instance of CypressTestExecutor
func NewCypressTestExecutor(
	log *slog.Logger,
	targetDir string,
	prepareCmd string,
	executeCmd string,
) *CypressTestExecutor {
	return &CypressTestExecutor{
		Log:        log,
		PrepareCmd: prepareCmd,
		ExecuteCmd: executeCmd,
		TargetDir:  targetDir,
	}
}

func (t *CypressTestExecutor) Name() string {
	return "Cypress"
}

// ExecTestSuite runs a test suite using cypress
// Can be used with the following commands
//
// go run . test --test-type smoke --test-suite e2e --runner cypress --pw-prepare-cmd "yarn install" --pw-execute-cmd "yarn e2e:jira" --grafana-username e2e --grafana-password e2e --pw-target-dir ./test-repo/cypress-e2e
func (t *CypressTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.TargetDir == "" {
		return executor.SuiteRunSummary{}, fmt.Errorf("missing target directory. Please pass the relative path to the test suite directory using --pw-target-dir flag")
	}

	if t.PrepareCmd == "" {
		return executor.SuiteRunSummary{}, fmt.Errorf("missing prepare command. Please pass the command using the flag --pw-prepare-cmd 'yarn install'")
	}

	if t.ExecuteCmd == "" {
		return executor.SuiteRunSummary{}, fmt.Errorf("missing execute command. Please pass the command using the flag --pw-execute-cmd 'yarn test'")
	}

	workingDir := t.TargetDir + "/" + suite.Path

	err := t.prepareCodebase(workingDir, t.PrepareCmd)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(workingDir, t.ExecuteCmd)
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

func (t *CypressTestExecutor) executeTests(testingDir string, executeCmd string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		testRunCmd := exec.Command("bash", "-c", executeCmd)
		if err := utils.ExecStdout(testRunCmd); err != nil {
			return err
		}

		return nil
	})
}
