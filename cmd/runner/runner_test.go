package main

import (
	"bufio"
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

var grafanaBuildInfo = map[string]interface{} {
	"buildInfo": map[string]interface{} {
		"version": "10.x.0-test",
		"commit": "a3b9ec21db4e50a90e049132723af118dc3f39b3",
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
		_, err  := buff.ReadFrom(r.Body)
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
		rw.Header().Add("Set-Cookie","grafana_session=ffffffffffffffffffffffffffffffff; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax")

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

func WithInvalidUserCredentials() testRunnerOption {
	return func(t *TestRunner) error {
		t.GrafanaInstance.User = "invalid"
		t.GrafanaInstance.ServicePassword = "invalid"
		return nil
	}
}

func testRunnerForTesting(
	log *slog.Logger,
	testType TestType,
	tests []string,
	grafanaInstance *provisioner.VMInstance,
	opts...testRunnerOption,
) (*TestRunner, error) {
	tr := NewTestRunner(
		log,
		false,           // prevent k6 test output in test output
		"test",          // trigger
		testType,
		tests,
		"test",          // test suite version
		"",              // k6Cloud project
		"",              // k6Cloud token
		grafanaInstance,
		time.Second,     // grafana liveness probe timeout
		"local",         // machine spec
		"devel",         // bench revision
	)

	// apply options
	for _, opt := range opts {
		if err := opt(tr);  err != nil {
			return nil, err
		}
	}
	return tr, nil
}

const loginError = "Error logging into grafana instance"

func Test_Runner(t *testing.T) {
	t.Parallel()

	testCases := []struct{
		testCase  string
		options   []testRunnerOption
		testType  TestType
		tests     []string
		expectErr string
	}{
		{
			testCase: "passing test",
			testType: SmokeTest,
			tests:    []string{"k6tests/pass.js"},
		},
		{
			testCase: "failing test",
			testType: SmokeTest,
			tests:    []string{"k6tests/fail.js"},
		},
		{
			testCase:  "wrong credentials",
			options:   []testRunnerOption{
				WithInvalidUserCredentials(),
			},
			testType:  SmokeTest,
			tests:     []string{"k6tests/pass.js"},
			expectErr: loginError,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.testCase, func(t *testing.T){
			t.Parallel()

			// TODO: search output for expected messages
			logBuffer := bytes.Buffer{}
			log := slog.New(slog.NewTextHandler(bufio.NewWriter(&logBuffer), nil))

			grafanaMock := httptest.NewServer(http.HandlerFunc(grafanaMockHandler))
			grafanaInstance, _ := provisioner.NewReadOnlyGrafanaVM(grafanaMock.URL, "admin", "admin")

			// create test runner with test-specific options
			tr, err := testRunnerForTesting(
				log,
				tc.testType,
				tc.tests,
				grafanaInstance,
				tc.options...,
			)
			if err != nil {
				t.Fatalf("failed to setup test %v", err)
			}
			
			// execute test
			err = tr.Exec(context.TODO())

			if err == nil && tc.expectErr == "" {
				return
			}

			if err == nil && tc.expectErr != "" {
				t.Fatalf("should had failed with %q", tc.expectErr)
			}
			
			// FIXME: checking for specific error text is fragile.
			// The TestRunner should return different error types to facilitate validations
			if err != nil && (tc.expectErr == "" || !strings.Contains(err.Error(), tc.expectErr)) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}