package gotest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

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
			opts:  GoExecutorOptions{
				TestArgs: []string{"-tags", "goexecutor"},
			},
			suite: executor.TestSuite{
				Path: "./tests",
			},
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 4,
				TestsPassed:   3,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "TestFailing", Status: executor.TestFailed},
					{TestFile: "TestPassing1", Status: executor.TestPassed},
					{TestFile: "TestPassing2", Status: executor.TestPassed},
					{TestFile: "TestPassing3", Status: executor.TestPassed},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			goExec := NewGoExecutor(log, tc.opts)
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
