package reporter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// fakeNotifier collects the list of tests to be notified to each recipient
type fakeNotifier struct {
	notifications map[string][]string
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{
		notifications: map[string][]string{},
	}
}

func (f *fakeNotifier) Notify(
	ctx context.Context,
	recipient string,
	suiteRunId string,
	testRuns []executor.TestRun,
) error {
	for _, testRun := range testRuns {
		f.notifications[recipient] = append(f.notifications[recipient], testRun.TestFile)
	}
	return nil
}

func TestNotificationReporter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title       string
		options     []NotificationOption
		suiteRun    executor.SuiteRunSummary
		codeowners  string
		expected    map[string][]string
		expectedErr error
	}{
		{
			title: "notify failed test to global code owner",
			suiteRun: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{TestFolder: "test-suite", TestFile: "pass.js", Status: executor.TestPassed},
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners: "* @grafana-bench",
			expected: map[string][]string{
				"@grafana-bench": {"failed.js"},
			},
		},
		{
			title:   "notify all tests to global code owner",
			options: []NotificationOption{NotifyAll},
			suiteRun: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{TestFolder: "test-suite", TestFile: "pass.js", Status: executor.TestPassed},
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners: "* @grafana-bench",
			expected: map[string][]string{
				"@grafana-bench": {"pass.js", "failed.js"},
			},
		},
		{
			title: "no code owner for failed test",
			suiteRun: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{TestFolder: "test-suite/test-folder", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners: "another-folder @grafana-bench",
			expected:   map[string][]string{},
		},
		{
			title:   "notify only failed tests with code owner",
			options: []NotificationOption{NotifyAll},
			suiteRun: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{TestFolder: "test-suite/folder", TestFile: "failed.js", Status: executor.TestFailed},
					{TestFolder: "test-suite/another-folder", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners: "test-suite/folder @grafana-bench",
			expected: map[string][]string{
				"@grafana-bench": {"failed.js"},
			},
		},
		{
			title: "no codeowners file",
			suiteRun: executor.SuiteRunSummary{
				TestRuns: []executor.TestRun{
					{TestFolder: "test-suite", TestFile: "pass.js", Status: executor.TestPassed},
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners:  "",
			expectedErr: nil,
			expected:    map[string][]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			testSuiteBaseDir := t.TempDir()
			testSuiteDir := "test-suite"
			if err := os.MkdirAll(filepath.Join(testSuiteBaseDir, testSuiteDir), 0o755); err != nil {
				t.Fatal(err)
			}

			// write codeowners if not empty
			if tc.codeowners != "" {
				if err := os.WriteFile(filepath.Join(testSuiteBaseDir, "CODEOWNERS"), []byte(tc.codeowners), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			// create a test suite with a base dir in the test's tmp dir
			// so the code owners files are located there
			testSuite := executor.TestSuite{
				Name:    "test suite",
				BaseDir: testSuiteBaseDir,
				Path:    testSuiteDir,
			}

			// create test files
			for _, testRun := range tc.suiteRun.TestRuns {
				testFolder := filepath.Join(testSuite.BaseDir, testRun.TestFolder)
				if err := os.MkdirAll(testFolder, 0o755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(testFolder, testRun.TestFile), []byte{}, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			notifier := newFakeNotifier()
			reporter := NewNotificationReporter(notifier, tc.options...)

			err := reporter.Report(
				context.Background(),
				"123", // run id
				"456", // test suite run id
				testSuite,
				tc.suiteRun,
			)

			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			if tc.expectedErr != nil {
				return
			}

			if !reflect.DeepEqual(notifier.notifications, tc.expected) {
				t.Fatalf("expected %v got %v", tc.expected, notifier.notifications)
			}
		})
	}
}
