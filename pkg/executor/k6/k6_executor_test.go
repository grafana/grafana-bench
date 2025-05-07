package k6

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"

)

type k6TestExecutorOption func(*K6TestExecutor) error

// configure TestRunner with retries
func WithRetries(retries int) k6TestExecutorOption {
	return func(t *K6TestExecutor) error {
		t.RetryFailed = retries
		return nil
	}
}

// configure TestRunner with cloud output
func WithCloudOutput() k6TestExecutorOption {
	return func(t *K6TestExecutor) error {
		t.CloudOutput = true
		return nil
	}
}

// configure TestRunner with K6 credentials
func WithK6Credentials() k6TestExecutorOption {
	return func(t *K6TestExecutor) error {
		t.CloudProjectID = "PROJECT_ID"
		t.CloudToken = "TOKEN"
		return nil
	}
}

func k6TestRunnerForTesting(
	log *slog.Logger,
	opts ...k6TestExecutorOption,
) (*K6TestExecutor, error) {
	te := NewK6TestExecutor(
		log,
		K6ExecutorOptions{
			Verbose: true,
		},
	)

	// apply options
	for _, opt := range opts {
		if err := opt(te); err != nil {
			return nil, err
		}
	}
	return te, nil
}


func TestK6Executor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase      string
		k6options     []k6TestExecutorOption
		testSuite     string
		expectSummary *executor.SuiteRunSummary
		expectErr     error
	}{
		{
			testCase:  "passing test",
			testSuite: "k6tests/pass.js",
			expectSummary: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsPassed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "pass.js", Status: executor.TestPassed},
				},
			},
		},
		{
			testCase:  "failing test",
			testSuite: "k6tests/fail.js",
			expectSummary: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "fail.js", Status: executor.TestFailed},
				},
			},
		},
		{
			testCase:  "retry failing test",
			testSuite: "k6tests/fail.js",
			k6options: []k6TestExecutorOption{
				WithRetries(3),
			},
			expectSummary: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "fail.js", Status: executor.TestFailed},
				},
			},
		},
		{
			testCase:  "error test",
			testSuite: "k6tests/abort.js",
			expectSummary: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsError:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "abort.js", Status: executor.TestError},
				},
			},
		},
		{
			testCase:  "missing test",
			testSuite: "k6tests/missing.js",
			expectErr: testFilesError,
		},
		{
			testCase:  "test suite directory",
			testSuite: "k6tests/",
			expectSummary: &executor.SuiteRunSummary{
				TestsExecuted: 3,
				TestsError:    1,
				TestsFailed:   1,
				TestsPassed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "abort.js", Status: executor.TestError},
					{TestFile: "fail.js", Status: executor.TestFailed},
					{TestFile: "pass.js", Status: executor.TestPassed},
				},
			},
		},
		{
			testCase: "with k6 cloud config",
			k6options: []k6TestExecutorOption{
				WithCloudOutput(),
				WithK6Credentials(),
			},
			testSuite: "k6tests/pass.js",
		},
		{
			testCase: "invalid k6 cloud config",
			k6options: []k6TestExecutorOption{
				WithCloudOutput(),
			},
			testSuite: "k6tests/pass.js",
			expectErr: missingK6CloudConfigError,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()

			logBuffer := bytes.Buffer{}
			log := slog.New(slog.NewTextHandler(&logBuffer, nil))

			suite := executor.TestSuite{
				Path:     tc.testSuite,
				Revision: "test",
			}

			k6Executor, err := k6TestRunnerForTesting(log, tc.k6options...)
			if err != nil {
				t.Fatalf("failed to setup the k6 executor %v", err)
			}

			summary, err := k6Executor.ExecTestSuite(context.TODO(), suite, map[string]string{})

			if tc.expectErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectErr != nil && !errors.Is(err, tc.expectErr) {
				t.Fatalf("should had failed with '%v' got: '%v'", tc.expectErr, err)
			}

			if err = assert.SuiteSummaryEqual(tc.expectSummary, summary); err != nil {
				t.Fatalf("invalid summary %v", err)
			}
		})
	}
}
