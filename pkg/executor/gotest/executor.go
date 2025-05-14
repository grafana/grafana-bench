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
	// GoArgs arguments to pass to go test (e.g. []string{"-tags", "slow", "-race")
	GoArgs []string
	// TestAgs arguments to be passed to the test (using go test -args)
	TestArgs []string
	// retries for failed tests
	Retries int
}

// GoExecutor implements an TestExecutor for go tests
type GoExecutor struct {
	log      *slog.Logger
	goArgs   []string
	testArgs []string
	retries  int
}

func NewGoExecutor(log *slog.Logger, opts GoExecutorOptions) *GoExecutor {
	return &GoExecutor{
		log:      log,
		goArgs:   opts.GoArgs,
		testArgs: opts.TestArgs,
		retries:  opts.Retries,
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
	stdOut, err := runGoTest(ctx, e.log, suite.Path, e.goArgs, e.testArgs)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to execute tests  %w", err)
	}

	summary, err := ParseJsonOutput(stdOut)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to parse go test output %w", err)
	}

	if summary.TestsFailed > 0 && e.retries > 0 {
		for i, t := range summary.TestRuns {
			if t.Status != executor.TestFailed {
				continue
			}
			tr, err := retryTest(ctx, e.log, e.retries, t.TestFolder, e.goArgs, e.testArgs, t.TestFile)
			if err != nil {
				return executor.SuiteRunSummary{}, fmt.Errorf("failed to run go test %w", err)
			}
			if tr.Status == executor.TestPassed {
				summary.TestRuns[i] = tr
				summary.TestRuns[i].Status = executor.TestFlaky
				summary.TestsFailed--
				summary.TestsFlaky++
			}
		}
	}

	return summary, nil
}

func retryTest(
	ctx context.Context,
	log *slog.Logger,
	retries int,
	pattern string,
	args []string,
	testArgs []string,
	test string,
) (executor.TestRunSummary, error) {
	args = append(args, "--run", test)
	result, err := runGoTest(ctx, log, pattern, args, testArgs)
	if err != nil {
		return executor.TestRunSummary{}, fmt.Errorf("failed to execute go test %w", err)
	}

	testRuns, err := parseTestRuns(result)
	if err != nil {
		return executor.TestRunSummary{}, fmt.Errorf("failed to parse go test output %w", err)
	}

	return *testRuns[testkey{pkg: pattern, test: test}], nil
}

func runGoTest(ctx context.Context, log *slog.Logger, pattern string, goArgs []string, testArgs []string) (io.Reader, error) {
	cmdArgs := []string{"test"}
	cmdArgs = append(cmdArgs, goArgs...) // add test args (e.g. -tags)
	cmdArgs = append(cmdArgs, pattern)   // use suite path as package selection pattern

	// capture output in json format from stdout
	cmdArgs = append(cmdArgs, "-json")
	if len(testArgs) > 0 {
		cmdArgs = append(cmdArgs, "-args")
		cmdArgs = append(cmdArgs, testArgs...)
	}
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
