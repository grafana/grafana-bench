// Package gotest implements a go test runner
// This code in inspired by https://github.com/tailscale/tailscale/blob/main/cmd/testwrapper/testwrapper.go
package gotest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// GoExecutorOptions defines the options for a GoExecutor
type GoExecutorOptions struct {
	// TestArgs arguments to pass to go test (e.g. []string{"-tags", "slow", "-race")
	TestArgs []string
}

// GoExecutor implements an TestExecutor for go tests
type GoExecutor struct {
	log  *slog.Logger
	args []string
}

func NewGoExecutor(log *slog.Logger, opts GoExecutorOptions) *GoExecutor {
	return &GoExecutor{
		log:  log,
		args: opts.TestArgs,
	}
}

// Name returns the name of the executor
func (e *GoExecutor) Name() string {
	return "gotest"
}

// ExecTestSuite executes a test suite an reports the results
func (e *GoExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {

	stdOut, err := runGoTest(ctx, e.log, suite.Path, e.args)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to execute tests  %w", err)
	}

	summary, err := ParseJsonOutput(stdOut)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to parse go test output %w", err)
	}

	return summary, nil
}

func runGoTest(ctx context.Context, log *slog.Logger, path string, args []string) (io.Reader, error) {
	cmdArgs := []string{"test"}
	cmdArgs = append(cmdArgs, args...) // add test args (e.g. -tags)
	cmdArgs = append(cmdArgs, path)    // use suite path as package selection pattern

	// capture output in json format from stdout
	cmdArgs = append(cmdArgs, "-json")
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Env = os.Environ()
	stdErr := &bytes.Buffer{}
	stdOut := &bytes.Buffer{}
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	log.Debug("executing go test", "args", cmd.Args)
	err := cmd.Run()
	// if test fails, return code is 1
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		log.Debug("go test failed", "err", err, "output", cmd.Stderr)
		return nil, fmt.Errorf("failed to execute go test command %w", err)
	}

	return stdOut, nil
}
