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
	grafanaVersion       = "10.x.0-test"
	grafanaBuildInfo     = "{\"buildInfo\": {\"version\": \"" + grafanaVersion + "\", \"commit\": \"a3b9ec21db4e50a90e049132723af118dc3f39b3\", \"buildstamp\": 1705409435}}"
	grafana_session      = "ffffffffffffffffffffffffffffffff"
	session_cookie       = "grafana_session=" + grafana_session + "; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax"
	invalidUserMessage   = "{\"message\": \"Invalid username or password\"}"
	loggedInMessage      = "{\"message\":\"Logged in\", \"redirectUrl\":\"/\""
	loadingMessage       = "{\"message\":\"Your instance is loading, and will be ready shortly.\"}"
	loginDisabledMessage = "{\"message\":\"Bad request\",\"messageId\":\"auth.client.notConfigured\"}"
)

func loginHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Set-Cookie", session_cookie)
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(loggedInMessage))
}

func invalidLoginHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusUnauthorized)
	rw.Write([]byte(invalidUserMessage))
}

func loginDisbledHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(loginDisabledMessage))
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
		{
			testCase:  "login disabled",
			mock:      newGrafanaMock(WithResponse("POST", "/login", loginDisbledHandler)),
			expectErr: LoginDisableError,
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

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expected    string
		expectError bool
	}{
		// Valid addresses with explicit scheme and port
		{
			name:     "https with explicit port",
			address:  "https://instance:3000",
			expected: "https://instance:3000",
		},
		{
			name:     "http with explicit port",
			address:  "http://instance:8080",
			expected: "http://instance:8080",
		},
		{
			name:     "https with explicit port and path",
			address:  "https://instance:3000/grafana",
			expected: "https://instance:3000/grafana",
		},
		{
			name:     "http with explicit port and path",
			address:  "http://instance:8080/grafana/dashboard",
			expected: "http://instance:8080/grafana/dashboard",
		},

		// Valid addresses with scheme, no port (port inferred)
		{
			name:     "https without port",
			address:  "https://instance",
			expected: "https://instance:443",
		},
		{
			name:     "http without port",
			address:  "http://instance",
			expected: "http://instance:80",
		},
		{
			name:     "https without port with path",
			address:  "https://instance/grafana",
			expected: "https://instance:443/grafana",
		},
		{
			name:     "http without port with path",
			address:  "http://instance/grafana/api",
			expected: "http://instance:80/grafana/api",
		},

		// Path variations
		{
			name:     "root path",
			address:  "https://instance:3000/",
			expected: "https://instance:3000/",
		},
		{
			name:     "nested path",
			address:  "https://instance:3000/grafana/d/dashboard",
			expected: "https://instance:3000/grafana/d/dashboard",
		},
		{
			name:     "path with query parameters",
			address:  "https://instance:3000/grafana?org=1",
			expected: "https://instance:3000/grafana",
		},

		// Domain variations
		{
			name:     "localhost",
			address:  "http://localhost:3000/grafana",
			expected: "http://localhost:3000/grafana",
		},
		{
			name:     "FQDN",
			address:  "https://grafana.example.com:3000/api",
			expected: "https://grafana.example.com:3000/api",
		},
		{
			name:     "IP address",
			address:  "http://192.168.1.100:3000/grafana",
			expected: "http://192.168.1.100:3000/grafana",
		},

		// Error cases
		{
			name:        "non-standard port without scheme",
			address:     "instance:3000",
			expectError: true,
		},
		{
			name:        "non-standard port with path without scheme",
			address:     "instance:3000/grafana",
			expectError: true,
		},
		{
			name:        "no port no scheme",
			address:     "instance",
			expectError: true,
		},
		{
			name:        "no port no scheme with path",
			address:     "instance/grafana",
			expectError: true,
		},
		{
			name:        "invalid scheme",
			address:     "ftp://instance:21",
			expectError: true,
		},
		{
			name:        "malformed URL",
			address:     "://invalid",
			expectError: true,
		},
		{
			name:        "empty address",
			address:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAddress(tt.address)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for address %q, but got none", tt.address)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for address %q: %v", tt.address, err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("parseAddress(%q) = %q, want %q", tt.address, result.String(), tt.expected)
			}
		})
	}
}

func TestParseAddressComponents(t *testing.T) {
	tests := []struct {
		name    string
		address string
		scheme  string
		host    string
		port    string
		path    string
	}{
		{
			name:    "full address with path",
			address: "https://grafana.example.com:3000/grafana/dashboard",
			scheme:  "https",
			host:    "grafana.example.com",
			port:    "3000",
			path:    "/grafana/dashboard",
		},
		{
			name:    "address without path",
			address: "http://localhost:8080",
			scheme:  "http",
			host:    "localhost",
			port:    "8080",
			path:    "",
		},
		{
			name:    "inferred port",
			address: "https://example.com/api",
			scheme:  "https",
			host:    "example.com",
			port:    "443",
			path:    "/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAddress(tt.address)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Scheme != tt.scheme {
				t.Errorf("scheme: got %q, want %q", result.Scheme, tt.scheme)
			}
			if result.Hostname() != tt.host {
				t.Errorf("host: got %q, want %q", result.Hostname(), tt.host)
			}
			if result.Port() != tt.port {
				t.Errorf("port: got %q, want %q", result.Port(), tt.port)
			}
			if result.Path != tt.path {
				t.Errorf("path: got %q, want %q", result.Path, tt.path)
			}
		})
	}
}

func TestParseAddressEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expected    string
		expectError bool
	}{
		{
			name:     "path with trailing slash",
			address:  "https://instance:3000/grafana/",
			expected: "https://instance:3000/grafana/",
		},
		{
			name:     "path with multiple slashes",
			address:  "https://instance:3000/grafana//dashboard",
			expected: "https://instance:3000/grafana//dashboard",
		},
		{
			name:     "path with encoded characters",
			address:  "https://instance:3000/grafana%20dashboard",
			expected: "https://instance:3000/grafana%20dashboard",
		},
		{
			name:     "IPv6 address",
			address:  "http://[::1]:3000/grafana",
			expected: "http://[::1]:3000/grafana",
		},
		{
			name:     "port with leading zeros",
			address:  "https://instance:0443/grafana",
			expected: "https://instance:0443/grafana",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAddress(tt.address)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for address %q, but got none", tt.address)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for address %q: %v", tt.address, err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("parseAddress(%q) = %q, want %q", tt.address, result.String(), tt.expected)
			}
		})
	}
}
