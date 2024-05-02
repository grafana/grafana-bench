package playwright

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

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

	// When run in the docker image the build path is set to the env var BENCH_PLAYWRIGHT_BUILD_PATH
	// otherwise will use the default path of repository
	playwrightBuildPath := os.Getenv("BENCH_PLAYWRIGHT_BUILD_PATH")
	if playwrightBuildPath == "" {
		playwrightBuildPath = "cmd/test/playwright/build"
	}

	fmt.Println("Running Playwright tests", "suite", suite.Path, "url", env["GRAFANA_URL"], playwrightplaywrightBuildPath)
	err := t.executeTests(suite.Path, env["GRAFANA_URL"])
	if err != nil {
		// process might return exit code 1 if test fails but we still want to try to parse the report
		t.Log.Info("Playwright test execution failed", "error", err.Error())
	}

	fmt.Println("executeTest output", err)

	file, err := os.ReadFile(playwrightBuildPath + "/" + jsonOutputName)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing playwright report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}

func (t *PlaywrightTestExecutor) executeTests(testingDir, grafana_url, playwrightBuildPath string) error {
	return utils.ExecuteInDir(playwrightBuildPath, func() error {
		// this sets the node_modules path to the current directory so that it can find the playwright package
		// when running the tests in other directory
		os.Setenv("NODE_PATH", playwrightBuildPath+"/node_modules")
		// this sets the url of the tests to point to what is passed in
		// no cli command flag to change via playwright cli
		// therefore updates values in playwright.config.js file
		os.Setenv("PLAYWRIGHT_BASE_URL", grafana_url)
		// this is the relative path to the test suite from our current project
		// no cli command flag to change via playwright cli
		// therefore updates values in playwright.config.js file
		os.Setenv("PLAYWRIGHT_TEST_DIR", testingDir)

		testRunCmd := exec.Command("yarn", "playwright", "test")

		return utils.ExecStdout(testRunCmd)
	})
}
