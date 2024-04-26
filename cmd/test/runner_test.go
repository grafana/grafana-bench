package test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
)

type dummyExecutor struct {
	summary executor.SuiteRunSummary
}

func (d dummyExecutor) Name() string {
	return "test-mock"
}

func (d dummyExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	return d.summary, nil
}

type mockGrafanaInstanceOption func(*mockGrafanaInstance)

func withGrafanaNotAlive() mockGrafanaInstanceOption {
	return func(m *mockGrafanaInstance) {
		m.err = grafana.InstanceNotAvailableError
	}
}

func WithInvalidGrafanaCredentials() mockGrafanaInstanceOption {
	return func(m *mockGrafanaInstance) {
		m.err = grafana.InvalidCredentialsError
	}
}

type mockGrafanaInstance struct {
	session  *http.Cookie
	address  string
	user     string
	password string
	err      error
	version  string
}

func (m *mockGrafanaInstance) Url() string {
	return m.address
}

func (m *mockGrafanaInstance) Password() string {
	return m.password
}

func (m *mockGrafanaInstance) UserName() string {
	return m.user
}

func (m *mockGrafanaInstance) WaitForLiveGrafana(_ context.Context) error {
	return m.err
}

func (m *mockGrafanaInstance) GetGrafanaBuildVersion() (string, error) {
	return m.version, m.err
}

func (m *mockGrafanaInstance) GetGrafanaSession() (string, error) {
	return m.session.Value, m.err
}

func newMockGrafanaInstance(opts ...mockGrafanaInstanceOption) *mockGrafanaInstance {
	mock := &mockGrafanaInstance{
		user:     "admin",
		password: "admin",
		err:      nil,
		version:  "test",
		session: &http.Cookie{
			Name:  "grafana_session",
			Value: "fake_grafana_session",
		},
	}

	for _, opFunc := range opts {
		opFunc(mock)
	}

	return mock
}

type testRunnerOption func(*TestRunner) error

// configure TestRunner with a dashboard URL template
func WithDashboard() testRunnerOption {
	return func(t *TestRunner) error {
		t.DashboardURL = "http://localhost/dashboard?env={{.SuiteRun}}"
		return nil
	}
}

// configure TestRunner with an invalid dashboard template
func WithInvalidDashboard() testRunnerOption {
	return func(t *TestRunner) error {
		t.DashboardURL = "http://localhost/dashboard?env={{.InvalidVar}}"
		return nil
	}
}

func testRunnerForTesting(
	log *slog.Logger,
	grafanaInstance grafana.GrafanaInstance,
	executor executor.TestExecutor,
	opts ...testRunnerOption,
) (*TestRunner, error) {
	tr := NewTestRunner(
		log,
		"test", // trigger
		grafanaInstance,
		"local", // machine spec
		"devel", // bench revision
		"",      // dashboard URL
		executor,
	)

	// apply options
	for _, opt := range opts {
		if err := opt(tr); err != nil {
			return nil, err
		}
	}
	return tr, nil
}

const (
	invalidDashboardMessage = "invalid template substitution"

	loginError = "Invalid credentials"

	grafanaNotAliveError = "Instance not available"

	testSuiteFailedError = "test suite failed: Too many test failures"

	testSuiteFailedDashboardError = `test suite failed: Too many test failures.*See dashboard`
)

func failedSuiteSummary() executor.SuiteRunSummary {
	return executor.SuiteRunSummary{
		StartTime:     time.Now(),
		TestsExecuted: 1,
		TestsFailed:   1,
		TestRuns: []executor.TestRun{
			{
				Status: executor.TestFailed,
				Order:  1,
			},
		},
	}
}

func passingSuiteSummary() executor.SuiteRunSummary {
	return executor.SuiteRunSummary{
		StartTime:     time.Now(),
		TestsExecuted: 1,
		TestsPassed:   1,
		TestRuns: []executor.TestRun{
			{
				Status: executor.TestPassed,
				Order:  1,
			},
		},
	}
}

func Test_Runner(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase   string
		instance   *mockGrafanaInstance
		options    []testRunnerOption
		summary    executor.SuiteRunSummary
		expectErr  string
		expectMsgs []string
	}{
		{
			testCase:   "passing suite",
			instance:   newMockGrafanaInstance(),
			summary:    passingSuiteSummary(),
			expectMsgs: []string{},
		},
		{
			testCase:   "failing suite without dashboard",
			instance:   newMockGrafanaInstance(),
			summary:    failedSuiteSummary(),
			expectErr:  testSuiteFailedError,
			expectMsgs: []string{},
		},
		{
			testCase: "failing suite with dashboard",
			instance: newMockGrafanaInstance(),
			summary:  failedSuiteSummary(),
			options: []testRunnerOption{
				WithDashboard(),
			},
			expectErr: testSuiteFailedDashboardError,
		},
		{
			testCase: "failing suite with invalid dashboard",
			instance: newMockGrafanaInstance(),
			summary:  failedSuiteSummary(),
			options: []testRunnerOption{
				WithInvalidDashboard(),
			},
			expectErr: testSuiteFailedError,
			expectMsgs: []string{
				invalidDashboardMessage,
			},
		},
		{
			testCase: "invalid credentials",
			instance: newMockGrafanaInstance(
				WithInvalidGrafanaCredentials(),
			),
			expectErr: loginError,
		},
		{
			testCase: "grafana not available",
			instance: newMockGrafanaInstance(
				withGrafanaNotAlive(),
			),
			expectErr: grafanaNotAliveError,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()

			logBuffer := bytes.Buffer{}
			log := slog.New(slog.NewTextHandler(&logBuffer, nil))

			dummyExecutor := dummyExecutor{summary: tc.summary}

			// create test runner with test-specific options
			tr, err := testRunnerForTesting(
				log,
				tc.instance,
				dummyExecutor,
				tc.options...,
			)
			if err != nil {
				t.Fatalf("failed to setup test runner %v", err)
			}

			suite := executor.TestSuite{
				Path:     "testsuite",
				Revision: "test",
			}

			// execute test
			err = tr.Exec(context.TODO(), SmokeTest, suite)

			if err == nil && tc.expectErr != "" {
				t.Fatalf("should had failed with %q", tc.expectErr)
			}

			errExp := regexp.MustCompile(tc.expectErr)
			if err != nil && !errExp.MatchString(err.Error()) {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, msg := range tc.expectMsgs {
				if !strings.Contains(logBuffer.String(), msg) {
					t.Log(logBuffer.String())
					t.Fatalf("should had reported: %q", msg)
				}
			}
		})
	}
}
