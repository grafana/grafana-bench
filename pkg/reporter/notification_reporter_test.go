package reporter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/notifier"
)

// fakeNotifier collects the list of tests to be notified to each recipient
type fakeNotifier struct {
	err           error
	notifications map[string][]string
}

func newFakeNotifier(err error) *fakeNotifier {
	return &fakeNotifier{
		err:           err,
		notifications: map[string][]string{},
	}
}

func (f *fakeNotifier) Notify(
	ctx context.Context,
	recipient string,
	suiteRunId string,
	testRuns []executor.TestRunSummary,
) error {
	if f.err != nil {
		return f.err
	}

	for _, testRun := range testRuns {
		f.notifications[recipient] = append(f.notifications[recipient], testRun.TestFile)
	}
	return nil
}

func TestNotificationReporter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title           string
		options         []NotificationOption
		notificationErr error
		summary         executor.SuiteRunSummary
		codeowners      string
		expected        map[string][]string
		expectedErr     error
	}{
		{
			title: "notify failed test to global code owner",
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
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
			options: []NotificationOption{NotifyPassing(true)},
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
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
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{TestFolder: "test-suite/test-folder", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners: "another-folder @grafana-bench",
			expected:   map[string][]string{},
		},
		{
			title: "notify only failed tests with code owner",
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
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
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners:  "",
			expectedErr: nil,
			expected:    map[string][]string{},
		},
		{
			title:           "error sending notification",
			notificationErr: errors.New("fake notification error"),
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners:  "* @grafana-bench",
			expectedErr: ErrSendingNotification,
			expected:    map[string][]string{},
		},
		{
			title:           "Ignore No mapping for recipient error",
			notificationErr: notifier.ErrNoMappingForCodeowner,
			summary: executor.SuiteRunSummary{
				TestRuns: []executor.TestRunSummary{
					{TestFolder: "test-suite", TestFile: "failed.js", Status: executor.TestFailed},
				},
			},
			codeowners:  "* @grafana-bench",
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
			for _, testRun := range tc.summary.TestRuns {
				testFolder := filepath.Join(testSuite.BaseDir, testRun.TestFolder)
				if err := os.MkdirAll(testFolder, 0o755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(testFolder, testRun.TestFile), []byte{}, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			notifier := newFakeNotifier(tc.notificationErr)
			reporter, err := NewNotificationReporter(testSuiteBaseDir, notifier, tc.options...)
			if err != nil {
				t.Fatalf("failed to create notification reporter: %v", err)
			}

			suiteRun := executor.SuiteRun{
				Name:          "test",
				Id:            "123",
				SuiteName:     testSuite.Name,
				SuiteRevision: testSuite.Revision,
			}
			err = reporter.Report(
				context.Background(),
				suiteRun,
				tc.summary,
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
