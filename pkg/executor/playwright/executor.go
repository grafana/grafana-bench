package playwright

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/pkg/executor"
)

const (
	ExecutorName = "playwright"
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
	return ExecutorName
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

	playwrightEnv := map[string]string{}
	playwrightEnv["path"] = os.Getenv("PATH")

	maps.Copy(playwrightEnv, env)

	// prepare test execution
	if t.PrepareCmd != "" {
		// allow multiple commands separated by ";"
		for _, cmd := range strings.Split(t.PrepareCmd, ";") {
			if cmd == "" {
				continue
			}
			if err := t.executeCommand(filepath.Join(suite.BaseDir, suite.Path), playwrightEnv, cmd); err != nil {
				return executor.SuiteRunSummary{}, fmt.Errorf("failed to prepare codebase: %w", err)
			}
		}
	}

	// create temporary file for test output
	jsonOutput, err := os.CreateTemp(os.TempDir(), "playwright-report-*.json")
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("creating report.json: %s", err.Error())
	}

	// execute tests in the test suite and redirect output to a json file
	// we assume here we can append the reporter and the test suite to the execute command
	// e.g yarn run test --reporter json tests/
	// set the output
	playwrightEnv["PLAYWRIGHT_JSON_OUTPUT_NAME"] = jsonOutput.Name()
	executeCmd := fmt.Sprintf("%s --reporter=json %s", t.ExecuteCmd, suite.Path)

	if err := t.executeCommand(filepath.Join(suite.BaseDir, suite.Path), playwrightEnv, executeCmd); err != nil {
		// we can't tell if there was a error executing the test or the test command was wrong (e.g. misspelled)
		// so we check if there's any report. If not, we assume the test was not executed and return
		// otherwise we are trying to process the report with parseJsonOutput below
		reportInfo, errStat := os.Stat(jsonOutput.Name())
		if errStat != nil || reportInfo.Size() == 0 {
			return executor.SuiteRunSummary{}, fmt.Errorf("error executing tests: %w", err)
		}
	}

	//parse output or report any problem
	return ParseJsonOutput(jsonOutput)
}

func (t *PlaywrightTestExecutor) executeCommand(execDir string, env map[string]string, cmd string) error {
	cmdFields := strings.Fields(cmd)

	execCmd := exec.Command(cmdFields[0], cmdFields[1:]...)
	execCmd.Dir = execDir

	// add env variables
	for key, value := range env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	buf := bytes.NewBuffer(nil)
	if t.Verbose {
		execCmd.Stdout = io.MultiWriter(buf, os.Stderr)
		execCmd.Stderr = io.MultiWriter(buf, os.Stderr)
	} else {
		execCmd.Stdout = buf
		execCmd.Stderr = buf
	}

	if err := execCmd.Run(); err != nil {
		// If we're in verbose mode, we will already have the error in the output
		// otherwise, print it now
		if !t.Verbose {
			fmt.Println("!verbose output:", buf.String())
		}

		return fmt.Errorf("error command failed: %w", err)
	}

	return nil
}
