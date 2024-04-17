package playwright

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/grafana/grafana-bench/cmd/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var targetDir = "./test-repo"
var jsonOutputName = "playwright-report.json"

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log *slog.Logger

	TestSuite     string
	TestSuiteRepo string
	PrepareCmd    string
	ExecuteCmd    string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	testSuiteRepo string,
	prepareCmd string,
	executeCmd string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:           log,
		TestSuiteRepo: testSuiteRepo,
		PrepareCmd:    prepareCmd,
		ExecuteCmd:    executeCmd,
	}
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// ExecTestSuite runs a test suite using playwright
// Can be used with the following command:
//
//		go run . test \
//		 --test-suite /path/to/test/folder \
//	  --test-type smoke \
//	  --runner playwright \
//	  --pw-prepareCmd "yarn install && yarn playwright install" \
//	  --pw-executeCmd "yarn playwright test" \
//	  --pw-test-suite-repo git@github.com:grafana/grafana-plugin-tests
//
// execute test suite
func (t *PlaywrightTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.TestSuiteRepo == "" {
		return executor.SuiteRunSummary{}, errors.New("missing test suite repository")
	}

	testingDir := utils.GetTestingDirectory(targetDir, t.TestSuiteRepo)

	tc := compile.NewTestCompiler(t.Log, testingDir, t.TestSuiteRepo, "")
	err := tc.CloneRepo(context.TODO())
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to cloning codebase: %s", err.Error())
	}

	err = t.prepareCodebase(testingDir, t.PrepareCmd)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %s", err.Error())
	}

	err = t.executeTests(testingDir, t.ExecuteCmd, suite.Path)
	if err != nil {
		// process might return exit code 1 if test fails but we still want to try to parse the report
		t.Log.Info("Playwright processes exited with code 1", "error", err.Error())
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/%s", testingDir, jsonOutputName))
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
	err := utils.ExecuteInDir(testingDir, func() error {
		prepareCmd := exec.Command("bash", "-c", prepareCmd)
		if err := utils.ExecStdout(prepareCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to installing dependencies: %s", err.Error())
	}

	return nil
}

func (t *PlaywrightTestExecutor) executeTests(testingDir, executeCmd string, testSuite string) error {
	err := utils.ExecuteInDir(testingDir, func() error {
		os.Setenv("PLAYWRIGHT_JSON_OUTPUT_NAME", jsonOutputName)
		cmd := strings.Fields(executeCmd)
		cmd = append(cmd, "--reporter", "json")

		if testSuite != "" {
			cmd = append(cmd, "--grep", testSuite)
		}

		testRunCmd := exec.Command(cmd[0], cmd[1:]...)
		if err := utils.ExecStdout(testRunCmd); err != nil {
			return err
		}

		return nil
	})

	return err
}
