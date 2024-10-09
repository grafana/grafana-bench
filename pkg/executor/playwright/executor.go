package playwright

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/pkg/executor"
)

const (
	chromiumPath = "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH"
)

var (
	errMissingExecuteCmd = errors.New("error missing --pw-execute-cmd from command line arguments ")
)

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log        *slog.Logger
	PrepareCmd string
	ExecuteCmd string
	Verbose    bool
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	verbose bool,
	prepareCmd string,
	executeCmd string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:        log,
		Verbose:    verbose,
		PrepareCmd: prepareCmd,
		ExecuteCmd: executeCmd,
	}
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// ExecTestSuite runs a test suite using playwright
func (t *PlaywrightTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.ExecuteCmd == "" {
		return executor.SuiteRunSummary{}, errMissingExecuteCmd
	}

	if os.Getenv(chromiumPath) == "" {
		t.Log.Warn("playwright configuration", "environment variable not set", chromiumPath)
	}

	playwrightEnv := map[string]string{}
	playwrightEnv["path"] = os.Getenv("PATH")
	playwrightEnv[chromiumPath] = os.Getenv(chromiumPath)
	for k, v := range env {
		playwrightEnv[k] = v
	}

	// prepare test execution
	if t.PrepareCmd != "" {
		if err := t.executeCommand(suite.BaseDir, playwrightEnv, t.PrepareCmd); err != nil {
			return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %w", err)
		}
	}

	// create temporary file for test output
	jsonOutputName := filepath.Join(os.TempDir(), "playwright-report-*.json")

	// execute tests in the test suite and redirect output to a json file
	// we assume here we can append the reporter and the test suite to the execute command
	// e.g yarn run test --reporter json tests/
	// set the output
	playwrightEnv["PLAYWRIGHT_JSON_OUTPUT_NAME"] = jsonOutputName
	executeCmd := fmt.Sprintf("%s --reporter=json %s", t.ExecuteCmd, suite.Path)

	if err := t.executeCommand(suite.BaseDir, playwrightEnv, executeCmd); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("error executing tests: %w", err)
	}

	file, err := os.ReadFile(jsonOutputName)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("error failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("error failed parsing playwright report: %w", err)
	}

	return runSummary, nil
}

func (t *PlaywrightTestExecutor) executeCommand(execDir string, env map[string]string, cmd string) error {
	cmdFields := strings.Fields(cmd)

	execCmd := exec.Command(cmdFields[0], cmdFields[1:]...)
	execCmd.Dir = execDir

	// add env variables
	for key, value := range env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	//fmt.Printf("\n cmd: %#v \n", execCmd)
	// capture output. Replicate to stdout/stderr if verbose mode
	buf := bytes.NewBuffer(nil)
	if t.Verbose {
		execCmd.Stdout = io.MultiWriter(buf, os.Stderr)
		execCmd.Stderr = io.MultiWriter(buf, os.Stderr)
	} else {
		execCmd.Stdout = buf
		execCmd.Stderr = buf
	}

	if err := execCmd.Run(); err != nil {
		// FIXME is this logic correct and should it go here???
		if !t.Verbose {
			fmt.Println("!verbose output:", buf.String())
		}

		return fmt.Errorf("error command failed: %w", err)
	}

	return nil
}
