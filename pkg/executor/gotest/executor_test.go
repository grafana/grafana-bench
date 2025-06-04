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
	"github.com/grafana/grafana-bench/pkg/utils/test/sort"
)

// TestExecutor executes the go test under ./tests folder and collects the execution summary.
// These tests include passing, failing and skipped tests.
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
				Packages: []string{"./..."},
			},
			suite: executor.TestSuite{
				Path: "tests",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 7,
				TestsPassed:   6,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFailing", Status: executor.TestFailed},
					{TestFile: "TestPassing1", Status: executor.TestPassed},
					{TestFile: "TestPassing2", Status: executor.TestPassed},
					{TestFile: "TestPassing3", Status: executor.TestPassed},
					{TestFile: "TestPassing3/SubTest1", Status: executor.TestPassed},
					{TestFile: "TestPassing3/SubTest2", Status: executor.TestPassed},
					{TestFile: "TestPassing4", Status: executor.TestPassed},
				},
			},
		},
		{
			title: "run with package pattern",
			opts: GoExecutorOptions{
				Packages: []string{"./subpkg/."},
			},
			suite: executor.TestSuite{
				Path: "tests",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 1,
				TestsPassed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestPassing4", Status: executor.TestPassed},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			opts := tc.opts

			opts.GoArgs = append(opts.GoArgs, "-tags", "goexecutor")

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			goExec := NewGoExecutor(log, opts)
			summary, err := goExec.ExecTestSuite(context.TODO(), tc.suite, map[string]string{})
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected %v got %v", tc.expectErr, err)
			}

			// we can't assert durations because are unpredictable
			assert.Equal(t, "tests executed", tc.expect.TestsExecuted, summary.TestsExecuted)
			assert.Equal(t, "tests passed", tc.expect.TestsPassed, summary.TestsPassed)
			assert.Equal(t, "tests error", tc.expect.TestsError, summary.TestsError)
			assert.Equal(t, "tests failed", tc.expect.TestsFailed, summary.TestsFailed)
			assert.Equal(t, "test runs len", len(tc.expect.TestRuns), len(summary.TestRuns))

			sort.SortTestRunByFilename(tc.expect.TestRuns)
			sort.SortTestRunByFilename(summary.TestRuns)
			for i, tr := range tc.expect.TestRuns {
				assert.Equal(t, "test file", tr.TestFile, summary.TestRuns[i].TestFile)
				assert.Equal(t, "test status", tr.Status, summary.TestRuns[i].Status)

			}
		})
	}
}

// TestFlakyTest test the retries of flaky tests. It executes the tests under the ./flaky folder
func TestFlakyTest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		opts      GoExecutorOptions
		suite     executor.TestSuite
		expectErr error
		expect    executor.SuiteRunSummary
	}{
		{
			title: "run flaky test",
			opts: GoExecutorOptions{
				Packages: []string{"./..."},
			},
			suite: executor.TestSuite{
				Path: "flaky",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFlaky", Status: executor.TestFailed},
				},
			},
		},
		{
			title: "retry flaky test",
			opts: GoExecutorOptions{
				Packages: []string{"./..."},
				Retries:  1,
			},
			suite: executor.TestSuite{
				Path: "flaky",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 1,
				TestsFlaky:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFlaky", Status: executor.TestFlaky},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			opts := tc.opts

			opts.GoArgs = append(opts.GoArgs, "-tags", "goexecutor")

			// creates tmp file used by TestFlaky as a mark for failing/passing.
			// See comment on flaky/flaky_test.go for details
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

			assert.SuiteSummaryEqual(t, &tc.expect, summary)
		})
	}
}
