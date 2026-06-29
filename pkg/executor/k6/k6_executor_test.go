package k6

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

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

// configure TestRunner with retry delay
func WithRetryDelay(d time.Duration) k6TestExecutorOption {
	return func(t *K6TestExecutor) error {
		t.RetryDelay = d
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

func TestHasK6ScriptError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		output string
		expect bool
	}{
		{
			name:   "logfmt stacktrace",
			output: `time="..." level=error msg="TypeError: ..." executor=per-vu-iterations scenario=default source=stacktrace`,
			expect: true,
		},
		{
			name:   "json stacktrace",
			output: `{"level":"error","msg":"TypeError: ...","source":"stacktrace"}`,
			expect: true,
		},
		{
			name:   "clean passing output",
			output: "running (00m00.0s), 0/1 VUs, 1 complete and 0 interrupted iterations",
			expect: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "has script error", tc.expect, hasK6ScriptError([]byte(tc.output)))
		})
	}
}

func TestK6Executor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase  string
		k6options []k6TestExecutorOption
		testSuite string
		expect    *executor.SuiteRunSummary
		expectErr error
	}{
		{
			testCase:  "passing test",
			testSuite: "k6tests/pass.js",
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsPassed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "pass.js", Status: executor.TestPassed, Attempts: 1, MaxAttempts: 1},
				},
			},
		},
		{
			testCase:  "failing test",
			testSuite: "k6tests/fail.js",
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "fail.js", Status: executor.TestFailed, Attempts: 1, MaxAttempts: 1},
				},
			},
		},
		{
			testCase:  "retry failing test",
			testSuite: "k6tests/fail.js",
			k6options: []k6TestExecutorOption{
				WithRetries(3),
			},
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "fail.js", Status: executor.TestFailed, Attempts: 4, MaxAttempts: 4},
				},
			},
		},
		{
			testCase:  "retry delay sleeps between attempts",
			testSuite: "k6tests/fail.js",
			k6options: []k6TestExecutorOption{
				WithRetries(1),
				WithRetryDelay(500 * time.Millisecond),
			},
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsFailed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "fail.js", Status: executor.TestFailed, Attempts: 2, MaxAttempts: 2},
				},
			},
		},
		{
			testCase:  "error test",
			testSuite: "k6tests/abort.js",
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsError:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "abort.js", Status: executor.TestError, Attempts: 1, MaxAttempts: 1},
				},
			},
		},
		{
			// k6 exits 0 on an uncaught script exception; bench must still
			// surface it as an error rather than a pass.
			testCase:  "script exception test",
			testSuite: "k6tests/exception.js",
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsError:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "exception.js", Status: executor.TestError, Attempts: 1, MaxAttempts: 1},
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
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 4,
				TestsError:    2,
				TestsFailed:   1,
				TestsPassed:   1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "abort.js", Status: executor.TestError, Attempts: 1, MaxAttempts: 1},
					{TestFile: "exception.js", Status: executor.TestError, Attempts: 1, MaxAttempts: 1},
					{TestFile: "fail.js", Status: executor.TestFailed, Attempts: 1, MaxAttempts: 1},
					{TestFile: "pass.js", Status: executor.TestPassed, Attempts: 1, MaxAttempts: 1},
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
			expect: &executor.SuiteRunSummary{
				TestsExecuted: 1,
				TestsError:    1,
				TestRuns: []executor.TestRunSummary{
					{TestFile: "pass.js", Status: executor.TestError, Attempts: 1, MaxAttempts: 1},
				},
			},
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

			start := time.Now()
			summary, err := k6Executor.ExecTestSuite(context.TODO(), suite, map[string]string{})
			elapsed := time.Since(start)

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected '%v' got: '%v'", tc.expectErr, err)
			}

			if tc.expectErr != nil {
				return
			}

			// when retry delay is set, wall-clock should be >= delay * (retries)
			if k6Executor.RetryDelay > 0 {
				minSleep := k6Executor.RetryDelay
				if elapsed < minSleep {
					t.Errorf("elapsed %v should be >= RetryDelay %v", elapsed, minSleep)
				}
			}

			// we can't assert durations because are unpredictable
			assert.Equal(t, "tests executed", tc.expect.TestsExecuted, summary.TestsExecuted)
			assert.Equal(t, "tests passed", tc.expect.TestsPassed, summary.TestsPassed)
			assert.Equal(t, "tests error", tc.expect.TestsError, summary.TestsError)
			assert.Equal(t, "tests failed", tc.expect.TestsFailed, summary.TestsFailed)
			assert.Equal(t, "test runs len", len(tc.expect.TestRuns), len(summary.TestRuns))
			for i, tr := range tc.expect.TestRuns {
				assert.Equal(t, "test file", tr.TestFile, summary.TestRuns[i].TestFile)
				assert.Equal(t, "test status", tr.Status, summary.TestRuns[i].Status)
				assert.Equal(t, "test attempts", tr.Attempts, summary.TestRuns[i].Attempts)
				assert.Equal(t, "test maxAttempts", tr.MaxAttempts, summary.TestRuns[i].MaxAttempts)
			}
		})
	}
}

// TestGetJsonOutputFilename verifies retries target a separate JSON file so
// prior attempts are preserved for postmortem.
func TestGetJsonOutputFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		filename string
		attempt  int
		want     string
	}{
		{"dashboard_create.js", 1, "/tmp/dashboard_create.json"},
		{"dashboard_create.js", 2, "/tmp/dashboard_create-attempt-2.json"},
		{"dashboard_create.js", 3, "/tmp/dashboard_create-attempt-3.json"},
		{"/some/abs/path/foo.js", 1, "/tmp/foo.json"},
		{"/some/abs/path/foo.js", 4, "/tmp/foo-attempt-4.json"},
	}
	for _, c := range cases {
		got := getJsonOutputFilename(c.filename, c.attempt)
		if got != c.want {
			t.Errorf("getJsonOutputFilename(%q, %d) = %q, want %q", c.filename, c.attempt, got, c.want)
		}
	}
}
