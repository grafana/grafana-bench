package playwright

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var jsonOutputName = "playwright-report.json"

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log *slog.Logger

	PrepareCmd string
	ExecuteCmd string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	prepareCmd string,
	executeCmd string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:        log,
		PrepareCmd: prepareCmd,
		ExecuteCmd: executeCmd,
	}
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// ExecTestSuite runs a test suite using playwright
// Can be used with the following commands
//
//	go run . test --test-suite smoke --test-type smoke --runner playwright --pw-prepare-cmd "yarn install && yarn playwright install" --pw-execute-cmd "yarn playwright test" --pw-target-dir ./test-repos/plugin-tests
//
// execute test suite
func (t *PlaywrightTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if suite.Path == "" {
		return executor.SuiteRunSummary{}, fmt.Errorf("missing target directory. Please pass the relative path to the test suite directory using --test-suite flag")
	}

	if t.ExecuteCmd == "" {
		return executor.SuiteRunSummary{}, fmt.Errorf("missing execute command. Please pass the command using the flag --pw-execute-cmd 'yarn test'")
	}

	err := t.prepareCodebase(suite.Path, t.PrepareCmd)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(suite.Path, t.ExecuteCmd, suite.Path)
	if err != nil {
		// process might return exit code 1 if test fails but we still want to try to parse the report
		t.Log.Info("Playwright test execution failed", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/%s", suite.Path, jsonOutputName))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing playwright report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}

func (t *PlaywrightTestExecutor) prepareCodebase(testingDir string, prepareCmd string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		prepareCmd := exec.Command("bash", "-c", prepareCmd)
		if err := utils.ExecStdout(prepareCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		return nil
	})
}

func (t *PlaywrightTestExecutor) executeTests(testingDir, executeCmd string, testSuite string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		os.Setenv("PLAYWRIGHT_JSON_OUTPUT_NAME", jsonOutputName)
		cmd := strings.Fields(executeCmd)
		cmd = append(cmd, "--reporter", "json")

		if testSuite != "" {
			cmd = append(cmd, "--grep", testSuite)
		}

		testRunCmd := exec.Command(cmd[0], cmd[1:]...)

		return utils.ExecStdout(testRunCmd)
	})
}
