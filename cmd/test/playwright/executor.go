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

	GrafanaUrl string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	prepareCmd string,
	executeCmd string,
	grafanaUrl string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:        log,
		PrepareCmd: prepareCmd,
		ExecuteCmd: executeCmd,
		GrafanaUrl: grafanaUrl,
	}
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// ExecTestSuite runs a test suite using playwright
// Can be used with the following commands
//
//	bench test --test-suite /home/bench/work/grafana-plugin-tests/ --test-type smoke --runner playwright --pw-prepare-cmd "yarn install" --pw-execute-cmd "yarn test" --grafana-url "http://host.docker.internal:3000"
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

	t.setupEnvironmentVariables()

	err = t.executeTests(suite.Path, t.ExecuteCmd)
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
		prepareCmd := exec.Command("sh", "-c", prepareCmd)
		if err := utils.ExecStdout(prepareCmd); err != nil {
			return fmt.Errorf("preparing test execution: %w", err)
		}

		return nil
	})
}

// setupEnvironmentVariables sets up the environment variables required for playwright tests
// These should be picked up in the playwright.config.ts file of the tests themselves
func (t *PlaywrightTestExecutor) setupEnvironmentVariables() {
	executablePath := os.Getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH")
	if executablePath == "" {
		t.Log.Warn("PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH environment variable is not set, playwright executor expects a path to the chromium executable.")
	}

	if t.GrafanaUrl != "" {
		os.Setenv("PLAYWRIGHT_BASE_URL", t.GrafanaUrl)
	} else {
		t.Log.Info("grafanaUrl is empty, defaulting to baseUrl found in playwright.config.ts")
	}
}

func (t *PlaywrightTestExecutor) executeTests(testingDir, executeCmd string) error {
	return utils.ExecuteInDir(testingDir, func() error {
		os.Setenv("PLAYWRIGHT_JSON_OUTPUT_NAME", jsonOutputName)
		cmd := strings.Fields(executeCmd)
		cmd = append(cmd, "--reporter", "json")

		testRunCmd := exec.Command(cmd[0], cmd[1:]...)

		return utils.ExecStdout(testRunCmd)
	})
}
