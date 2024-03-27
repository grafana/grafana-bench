package grafana

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	grafanaVersion     = "10.x.0-test"
	grafanaBuildInfo   = "{\"buildInfo\": {\"version\": \"" + grafanaVersion + "\", \"commit\": \"a3b9ec21db4e50a90e049132723af118dc3f39b3\", \"buildstamp\": 1705409435}}"
	grafana_session    = "ffffffffffffffffffffffffffffffff"
	session_cookie     = "grafana_session=" + grafana_session + "; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax"
	invalidUserMessage = "{\"message\": \"Invalid username or password\"}"
	loggedInMessage    = "{\"message\":\"Logged in\", \"redirectUrl\":\"/\""
	loadingMessage     = "{\"message\":\"Your instance is loading, and will be ready shortly.\"}"
)

type response struct {
	Status  int
	Body    string
	Headers map[string]string
}

func loginHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Set-Cookie", session_cookie)
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(loggedInMessage))
}

func invalidLoginHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusUnauthorized)
	rw.Write([]byte(invalidUserMessage))
}

func serverErrorHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusInternalServerError)
}

func buildInfoHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusOK)
	rw.Header().Add("Content-Type", "application/json")
	rw.Write([]byte(grafanaBuildInfo))
}

type routerOption func(r *httprouter.Router)

func WithResponse(method string, route string, handler http.HandlerFunc) routerOption {
	return func(m *httprouter.Router) {
		m.HandlerFunc(method, route, handler)
	}
}

// register a handler that returns 503 until a delay has passed
func With503Response(delay time.Duration, method string, route string, handler http.HandlerFunc) routerOption {
	return func(m *httprouter.Router) {
		deadline := time.Now().Add(delay)
		m.HandlerFunc(method, route, func(rw http.ResponseWriter, r *http.Request) {
			if time.Now().Before(deadline) {
				rw.WriteHeader(http.StatusServiceUnavailable)
				rw.Write([]byte(loadingMessage))
				return
			}

			handler(rw, r)
		})
	}
}

func newGrafanaMock(options ...routerOption) *httprouter.Router {
	// set default responses
	mock := httprouter.New()

	for _, optFunc := range options {
		optFunc(mock)
	}
	return mock
}

func Test_GetGrafanaSession(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase  string
		mock      *httprouter.Router
		expectErr error
	}{
		{
			testCase: "valid credentials",
			mock: newGrafanaMock(
				WithResponse("POST", "/login", loginHandler),
			),
			expectErr: nil,
		},
		{
			testCase:  "invalid credentials",
			mock:      newGrafanaMock(WithResponse("POST", "/login", invalidLoginHandler)),
			expectErr: InvalidCredentialsError,
		},
		{
			testCase:  "server error",
			mock:      newGrafanaMock(WithResponse("POST", "/login", serverErrorHandler)),
			expectErr: FailedRequestError,
		},
		{
			testCase:  "server loading",
			mock:      newGrafanaMock(With503Response(3*time.Second, "POST", "/login", loginHandler)),
			expectErr: nil,
		},
		{
			testCase:  "timeout waiting server",
			mock:      newGrafanaMock(With503Response(5*time.Second, "POST", "/login", loginHandler)),
			expectErr: InstanceNotAvailableError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(tc.mock)
			instance, err := NewInstance(mockServer.URL, "admin", "admin", WithTimeout(time.Second*3))
			if err != nil {
				t.Fatalf("unexpected error in test setup %v", err)
			}

			session, err := instance.GetGrafanaSession()

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("unexpected error expected: '%v' got '%v'", tc.expectErr, err)
			}

			if tc.expectErr == nil && session != grafana_session {
				t.Fatalf("invalid session expected %q got %q", grafana_session, session)
			}

		})
	}
}

func Test_GetGrafanaBuildVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase  string
		mock      *httprouter.Router
		expectErr error
	}{
		{
			testCase: "valid credentials",
			mock: newGrafanaMock(
				WithResponse("POST", "/login", loginHandler),
				WithResponse("GET", "/api/frontend/settings", buildInfoHandler),
			),
			expectErr: nil,
		},
		{
			testCase: "invalid credentials",
			mock: newGrafanaMock(
				WithResponse("POST", "/login", invalidLoginHandler),
				WithResponse("GET", "/api/frontend/settings", buildInfoHandler),
			),
			expectErr: InvalidCredentialsError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(tc.mock)
			instance, err := NewInstance(mockServer.URL, "admin", "admin")
			if err != nil {
				t.Fatalf("unexpected error in test setup %v", err)
			}

			version, err := instance.GetGrafanaBuildVersion()

			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("unexpected error expected: '%v' got '%v'", tc.expectErr, err)
			}

			if tc.expectErr == nil && version != grafanaVersion {
				t.Fatalf("invalid version expected %q got %q", grafanaVersion, version)
			}
		})
	}
}
