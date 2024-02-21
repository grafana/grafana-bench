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
		rw.Header().Add("Set-Cookie", "grafana_session=ffffffffffffffffffffffffffffffff; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax")

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

type testRunnerOption func(*TestRunner) error

// configure TestRunner with invalid grafana credentials
func WithInvalidGrafanaCredentials() testRunnerOption {
	return func(t *TestRunner) error {
		t.GrafanaInstance.User = "invalid"
		t.GrafanaInstance.ServicePassword = "invalid"
		return nil
	}
}

// configure TestRunner with cloud output
func WithCloudOutput() testRunnerOption {
	return func(t *TestRunner) error {
		t.K6CloudOutput = true
		return nil
	}
}

// configure TestRunner with invalid K6 credentials
func WithInvalidK6Credentials() testRunnerOption {
	return func(t *TestRunner) error {
		t.K6CloudProjectID = "INVALID_ID"
		t.K6CloudToken = "INVALID_TOKEN"
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
	testType TestType,
	testSuite string,
	grafanaInstance *provisioner.VMInstance,
	opts ...testRunnerOption,
) (*TestRunner, error) {
	tr := NewTestRunner(
		log,
		false,  // prevent k6 test output in test output
		false,  // prevent sending output to cloud
		"test", // trigger
		testType,
		"",        // base dir
		testSuite, // test suite path
		"test", // test suite name
		"test", // test suite version
		"",     // k6Cloud project
		"",     // k6Cloud token
		grafanaInstance,
		time.Second, // grafana liveness probe timeout
		"local",     // machine spec
		"devel",     // bench revision
		"",          // dashboard URL
	)

	// apply options
	for _, opt := range opts {
		if err := opt(tr); err != nil {
			return nil, err
		}
	}
	return tr, nil
}

const loginError = "Error logging into grafana instance"

const testSuiteMissingError = "not found"

const testSuiteFailedMessage = "test suite failed. Too many test failures"

const cloudOutputDisabledMessage = "running load tests with cloud output disabled"

const dashboardMessage = "See dashboard"

const invalidDashboardError = "invalid template substitution"

const cloudOutputParsingErrorMessage = "error parsing cloud run from K6 summary"

const missingK6CloudConfigError = "k6 Token and project ID are required for cloud output"

func Test_Runner(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase   string
		options    []testRunnerOption
		testType   TestType
		testSuite  string
		expectErr  string
		expectMsgs []string
	}{
		{
			testCase:  "passing test (load)",
			testType:  SmokeTest,
			testSuite: "k6tests/pass.js",
		},
		{
			testCase:   "passing test without k6 token (smoke)",
			testType:   LoadTest,
			testSuite:  "k6tests/pass.js",
			expectMsgs: []string{cloudOutputDisabledMessage},
		},
		{
			testCase: "load test without k6 config",
			options: []testRunnerOption{
				WithCloudOutput(),
			},
			testType:  LoadTest,
			testSuite:  "k6tests/pass.js",
			expectErr: missingK6CloudConfigError,
		},
		{
			testCase: "load test with invalid k6 config",
			options: []testRunnerOption{
				WithCloudOutput(),
				WithInvalidK6Credentials(),
			},
			testType:   LoadTest,
			testSuite:  "k6tests/pass.js",
			expectMsgs: []string{cloudOutputParsingErrorMessage},
		},
		{
			testCase:   "failing test (smoke)",
			testType:   SmokeTest,
			testSuite:  "k6tests/fail.js",
			expectMsgs: []string{testSuiteFailedMessage},
		},
		{
			testCase: "failing test with dashboard (smoke)",
			options: []testRunnerOption{
				WithDashboard(),
			},
			testType: SmokeTest,
			testSuite:  "k6tests/fail.js",
			expectMsgs: []string{
				testSuiteFailedMessage,
				dashboardMessage,
			},
		},
		{
			testCase: "failing test with invalid dashboard (smoke)",
			testType: SmokeTest,
			options: []testRunnerOption{
				WithInvalidDashboard(),
			},
			testSuite:  "k6tests/fail.js",
			expectErr: invalidDashboardError,
		},
		{
			testCase: "failing test (load)",
			testType: SmokeTest,
			testSuite:  "k6tests/fail.js",
		},
		{
			testCase:   "missing test (smoke)",
			testType:   SmokeTest,
			testSuite:  "k6tests/missing.js",
			expectErr:  testSuiteMissingError,
		},
		{
			testCase:   "test suite directory - one fails (smoke)",
			testType:   SmokeTest,
			testSuite:  "k6tests/",
			expectMsgs: []string{testSuiteFailedMessage},
		},
		{
			testCase: "wrong credentials",
			options: []testRunnerOption{
				WithInvalidGrafanaCredentials(),
			},
			testType:  SmokeTest,
			testSuite: "k6tests/pass.js",
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

			// create test runner with test-specific options
			tr, err := testRunnerForTesting(
				log,
				tc.testType,
				tc.testSuite,
				grafanaInstance,
				tc.options...,
			)
			if err != nil {
				t.Fatalf("failed to setup test %v", err)
			}

			// execute test
			err = tr.Exec(context.TODO())

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
