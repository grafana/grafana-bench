package test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
)

var grafanaBuildInfo = map[string]interface{}{
	"buildInfo": map[string]interface{}{
		"version":    "10.x.0-test",
		"commit":     "a3b9ec21db4e50a90e049132723af118dc3f39b3",
		"buildstamp": 1705409435,
	},
}

// Mocks grafana endpoints needed for the runner.
// Does not mock endpoints needed by tests!
func grafanaMockHandler(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/login":
		var loginInfo map[string]interface{}
		buff := bytes.Buffer{}
		_, err := buff.ReadFrom(r.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal(buff.Bytes(), &loginInfo)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// check for defaul user and password
		if loginInfo["user"] != "admin" || loginInfo["password"] != "admin" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		// return session cookie it is expected by VMInstance
		rw.Header().
			Add("Set-Cookie", "grafana_session=ffffffffffffffffffffffffffffffff; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax")

	// returns only the build info. TODO: add other attributes to response
	case "/api/frontend/settings":
		buff, err := json.Marshal(grafanaBuildInfo)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = rw.Write(buff)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Add("Content-Type", "application/json")
	default:
		rw.WriteHeader(http.StatusNotImplemented)
	}
}


type dummyExecutor struct {
	summary SuiteRunSummary
}

func (d dummyExecutor) Name() string {
	return "test-mock"
}

func (d dummyExecutor) 	ExecTestSuite(
	ctx context.Context,
	suite TestSuite,
	env map[string]string,
) (SuiteRunSummary, error) {
	return d.summary, nil
}


type testRunnerOption func(*TestRunner) error

// configure TestRunner with invalid grafana credentials
func WithInvalidGrafanaCredentials() testRunnerOption {
	return func(t *TestRunner) error {
		t.GrafanaInstance.User = "invalid"
		t.GrafanaInstance.ServicePassword = "invalid"
		return nil
	}
}

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
	grafanaInstance *provisioner.VMInstance,
	executor TestExecutor,
	opts ...testRunnerOption,
) (*TestRunner, error) {
	tr := NewTestRunner(
		log,
		"test",       // trigger
		grafanaInstance,
		time.Second, // grafana liveness probe timeout
		"local",     // machine spec
		"devel",     // bench revision
		"",          // dashboard URL
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
	loginError = "Error logging into grafana instance"

	dashboardMessage = "See dashboard"

	invalidDashboardError = "invalid template substitution"

 	testSuiteFailedMessage = "test suite failed. Too many test failures"
)

func failedSuiteSummary() SuiteRunSummary {
	return SuiteRunSummary{
		StartTime: time.Now(),
		TestsExecuted: 1,
		TestsFailed: 1,
		TestRuns: []TestRun{
			{ 
				Status: TestFailed,
				Order: 1,
			},
		},
	}
}

func passingSuiteSummary() SuiteRunSummary {
	return SuiteRunSummary{
		StartTime: time.Now(),
		TestsExecuted: 1,
		TestsPassed: 1,
		TestRuns: []TestRun{
			{ 
				Status: TestPassed,
				Order: 1,
			},
		},
	}
}

func Test_Runner(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase   string
		options    []testRunnerOption
		summary    SuiteRunSummary
		expectErr  string
		expectMsgs []string
	}{
		{
			testCase: "failing suite",
			summary: failedSuiteSummary(),
			expectMsgs: []string{testSuiteFailedMessage},
		},
		{
			testCase: "failing suite with dashboard",
			summary: failedSuiteSummary(),
			options: []testRunnerOption{
				WithDashboard(),
			},
			expectMsgs: []string{
				testSuiteFailedMessage,
				dashboardMessage,
			},
		},
		{
			testCase: "failing suite with invalid dashboard",
			summary: failedSuiteSummary(),
			options: []testRunnerOption{
				WithInvalidDashboard(),
			},
			expectErr: invalidDashboardError,
		},
		{
			testCase: "wrong credentials",
			options: []testRunnerOption{
				WithInvalidGrafanaCredentials(),
			},
			expectErr: loginError,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()

			logBuffer := bytes.Buffer{}
			log := slog.New(slog.NewTextHandler(&logBuffer, nil))

			grafanaMock := httptest.NewServer(http.HandlerFunc(grafanaMockHandler))
			grafanaInstance, _ := provisioner.NewReadOnlyGrafanaVM(grafanaMock.URL, "admin", "admin")

			executor := dummyExecutor{summary: tc.summary}

			// create test runner with test-specific options
			tr, err := testRunnerForTesting(
				log,
				grafanaInstance,
				executor,
				tc.options...,
			)
			if err != nil {
				t.Fatalf("failed to setup test runner %v", err)
			}

			suite := TestSuite{
				Path: "testsuite",
				Revision: "test",
			}

			// execute test
			err = tr.Exec(context.TODO(), SmokeTest, suite)

			if err == nil && tc.expectErr != "" {
				t.Fatalf("should had failed with %q", tc.expectErr)
			}

			// FIXME: checking for specific error text is fragile.
			// The TestRunner should return different error types to facilitate validations
			if err != nil && (tc.expectErr == "" || !strings.Contains(err.Error(), tc.expectErr)) {
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
