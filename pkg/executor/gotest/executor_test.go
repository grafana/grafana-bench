package gotest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

// TextExecutor executes the go test under ./tests folder and collects the execution summary.
// These tests include passing, failing, flaky and skipped tests.
func TestExecutor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		opts      GoExecutorOptions
		suite     executor.TestSuite
		expectErr error
		expect    executor.SuiteRunSummary
	}{

		{
			title: "run all tests",
			opts: GoExecutorOptions{
			},
			suite: executor.TestSuite{
				Path: "./tests",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 5,
				TestsPassed:   3,
				TestsFailed:   2,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFailing", Status: executor.TestFailed},
					{TestFile: "TestPassing1", Status: executor.TestPassed},
					{TestFile: "TestPassing2", Status: executor.TestPassed},
					{TestFile: "TestPassing3", Status: executor.TestPassed},
					{TestFile: "TestFlaky", Status: executor.TestFailed},
				},
			},
		},
		{
			title: "retry failed tests",
			opts: GoExecutorOptions{
				Retries: 1,
			},
			suite: executor.TestSuite{
				Path: "./tests",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 5,
				TestsPassed:   3,
				TestsFailed:   1,
				TestsFlaky:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFailing", Status: executor.TestFailed},
					{TestFile: "TestPassing1", Status: executor.TestPassed},
					{TestFile: "TestPassing2", Status: executor.TestPassed},
					{TestFile: "TestPassing3", Status: executor.TestPassed},
					{TestFile: "TestFlaky", Status: executor.TestFlaky},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			opts := tc.opts

			// The tests in ./test folder use the 'goexecutor' build tag to prevent running them as part of
			// bench's test, because some are designed to fail. This tag is set when invoking the executor
			opts.GoArgs = append(opts.GoArgs, "-tags", "goexecutor")

			// creates tmp file used by TestFlaky as a mark for failing/passing.
			// See comment on test/flaky_test.go for details
			flakyMark, err := os.CreateTemp(t.TempDir(), "flaky-mark-*")
			if err != nil {
				t.Fatalf("setting up flaky test mark file %v", err)
			}
			opts.TestArgs = append(opts.TestArgs, "-flaky-mark-file", flakyMark.Name())

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			goExec := NewGoExecutor(log, opts)
			summary, err := goExec.ExecTestSuite(context.TODO(), tc.suite, map[string]string{})
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}

			err = assert.SuiteSummaryEqual(&tc.expect, summary)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}
