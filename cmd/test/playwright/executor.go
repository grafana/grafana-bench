package playwright

import (
	"bytes"
	"context"
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
	chromiumPath    = "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH"
)

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log *slog.Logger
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
		return executor.SuiteRunSummary{}, fmt.Errorf("missing execute command.")
	}

	if os.Getenv(chromiumPath) ==  "" {
		t.Log.Warn("playwright configuration", "environment variable not set", chromiumPath)
	}

	// prepare test execution
	if t.PrepareCmd != "" {
		if err := t.executeCommand(suite.BaseDir, env, t.PrepareCmd); err != nil {
			return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %w", err)
		}
	}

	// create temporary file for test output
	jsonOutputName := filepath.Join(os.TempDir(), "playwright-report-*.json")

	// execute tests in the test suite and redirect output to a json file
	// we assume here we can append the reporter and the test suite to the execute command
	//
	// e.g yarn test --reporter json tests/
	// FIXME: we are modifying env. Maybe we should copy it
	env["PLAYWRIGHT_JSON_OUTPUT_NAME"] = jsonOutputName
	executeCmd := fmt.Sprintf(
		"%s --reporter json %s",
		t.ExecuteCmd,
		suite.Path,
	)

	if err := t.executeCommand(suite.BaseDir, env, executeCmd); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("executing tests %w", err)
	}

	file, err := os.ReadFile(jsonOutputName)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing playwright report: %w", err)
	}

	return runSummary, nil
}

func (t *PlaywrightTestExecutor) executeCommand(execDir string, env map[string]string, cmd string) error {
	cmdFields := strings.Fields(cmd)
	execCmd := exec.Command(cmdFields[0], cmdFields[1:]...)
	execCmd.Dir = execDir

	// capture output. Replicate to stdout/stderr if verbose mode
	buf := bytes.NewBuffer(nil)
	if t.Verbose {
		execCmd.Stdout = io.MultiWriter(buf, os.Stderr)
		execCmd.Stderr = io.MultiWriter(buf, os.Stderr)
	} else {
		execCmd.Stdout = buf
		execCmd.Stderr = buf
	}

	//set path
	execCmd.Env = append(execCmd.Env, os.Getenv("PATH"))

	// add env variables
	for key, value := range env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	if err := execCmd.Run(); err != nil {
		if!t.Verbose {
			fmt.Println(buf.String())
		}

		return fmt.Errorf("executing command %w", err)
	}

	return nil
}
