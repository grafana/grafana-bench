package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var grafanaBuildInfo = map[string]interface{}{
	"buildInfo": map[string]interface{}{
		"version":    "10.x.0-test",
		"commit":     "a3b9ec21db4e50a90e049132723af118dc3f39b3",
		"buildstamp": 1705409435,
	},
}

const grafana_session = "grafana_session=ffffffffffffffffffffffffffffffff; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax"

type grafanaMock struct {
	status   int
	user     string
	password string
	buildInfo map[string]interface{}
}

type grafanaMockOptions func(*grafanaMock)

// force return status
func WithStatus(status int) grafanaMockOptions {
	return func(m *grafanaMock) {
		m.status = status
	}
}

// change default credentials
func WithCredentials(user string, password string) grafanaMockOptions {
	return func(m *grafanaMock) {
		m.user = user
		m.password = password
	}
}
func newGrafanaMock(options...grafanaMockOptions) *grafanaMock {
	mock := &grafanaMock{
		user: "admin",
		password: "admin",
		status: http.StatusOK,
		buildInfo: grafanaBuildInfo,
	}

	for _, optionFunction := range options {
		optionFunction(mock)
	}

	return mock
}

// Mocks grafana endpoints needed for the runner.
// Does not mock endpoints needed by tests!
func (g *grafanaMock) Handler(rw http.ResponseWriter, r *http.Request) {
	if g.status != http.StatusOK {
		rw.WriteHeader(g.status)
		return
	}

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

		// check for default user and password
		if loginInfo["user"] != g.user || loginInfo["password"] != g.password {
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte("{\"message\": \"Invalid username or password\"}"))
			return
		}

		// return session cookie it is expected by VMInstance
		rw.Header().
			Add("Set-Cookie", grafana_session)
		rw.Write([]byte("{\"message\":\"Logged in\", \"redirectUrl\":\"/\""))

	// returns only the build info. TODO: add other attributes to response
	case "/api/frontend/settings":
		buff, err := json.Marshal(g.buildInfo)
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

type grafanaInstanceOption func(*grafanaInstance) error


// configure TestRunner with invalid grafana credentials
func WithInvalidGrafanaCredentials() grafanaInstanceOption {
	return func(g *grafanaInstance) error {
		g.user = "invalid"
		g.password = "invalid"
		return nil
	}
}

func Test_Login(t *testing.T) {
	t.Parallel()

	testCases := []struct{
		testCase  string
		mock      *grafanaMock
		expectErr error
	}{
		{
			testCase: "valid credentials",
			mock:  newGrafanaMock(),
			expectErr: nil,
		},
		{
			testCase: "invalid credentials",
			mock:  newGrafanaMock(WithCredentials("admin", "other password")),
			expectErr: InvalidCredentialsError,
		},
		{
			testCase: "server error",
			mock:  newGrafanaMock(WithStatus(http.StatusInternalServerError)),
			expectErr: FailedRequestError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(tc.mock.Handler))
			instance, err := NewInstance(mockServer.URL, "admin", "admin")
			if err != nil {
				t.Fatalf("unexpected error in test setup %v", err)
			}

			_, err = instance.GetGrafanaSession()

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("unexpected error expected: '%v' got '%v'", tc.expectErr, err)
			}
		})
	}
}
