package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GrafanaInstance represents an endpoint for accessing a Grafana Instance
type GrafanaInstance interface {
	// Address returns the url to access the grafana instance
	Address() string

	// UserName returns the user por accessing the instance
	UserName() string

	// Password returns the password for the grafana user
	Password() string

	// GetGrafanaBuildVersio returns the build version of the grafana instance
	GetGrafanaBuildVersion() (string, error)

	// GetGrafanaSession returns the current grafana session. Logs in if none is active
	GetGrafanaSession() (string, error)

	// WaitForLiveGrafana waits until the grafana instance is ready to accept requests
	WaitForLiveGrafana(context.Context) error
}

var (
	DefaultGrafanaTimeout     = time.Second * 60
	DefaultGrafanaBackoff     = time.Second
	FailedRequestError        = errors.New("Failed request")
	InvalidCredentialsError   = errors.New("Invalid credentials")
	InstanceNotAvailableError = errors.New("Instance not available")
	LoginDisableError         = errors.New("Login disabled")
)

type grafanaInstance struct {
	host        string
	address     string
	user        string
	password    string
	session     *http.Cookie
	timeout     time.Duration
	backoff     time.Duration
}

type grafanaInstanceOption func(*grafanaInstance) error

// Sets the grafana timeout. If 0, the default is used
func WithTimeout(timeout time.Duration) grafanaInstanceOption {
	return func(g *grafanaInstance) error {
		if timeout != 0 {
			g.timeout = timeout
		}
		return nil
	}
}

// Sets the grafana backoff time. If 0, the default is used
func WithBackoff(backoff time.Duration) grafanaInstanceOption {
	return func(g *grafanaInstance) error {
		if backoff != 0 {
			g.backoff = backoff
		}
		return nil
	}
}

// NewGrafanaInstance creates a reference to access a grafana instance
// Takes a fully qualified address such as https://jefflevinslunch.grafana.net
// and a user credentials
func NewInstance(address, user, password string, opts...grafanaInstanceOption) (GrafanaInstance, error) {
	scheme, host, port, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	instance := &grafanaInstance{
		host:     fmt.Sprintf("%s:%s", host, port),
		address:  fmt.Sprintf("%s://%s:%s", scheme, host, port),
		user:     user,
		password: password,
		timeout:  DefaultGrafanaTimeout,
		backoff:  DefaultGrafanaBackoff,
	}

	for _, optFunc := range opts {
		if err = optFunc(instance); err != nil {
			return nil, fmt.Errorf("invalid option %v", err)
		}
	}

	return instance, nil
}

// Address returns the url to access the grafana instance
func (g *grafanaInstance) Address() string {
	return g.address
}

// UserName returns the user por accessing the instance
func (g *grafanaInstance) UserName() string {
	return g.user
}

// Password returns the password for the grafana user
func (g *grafanaInstance) Password() string {
	return g.password
}

// GetSession returns the current grafana session value
func (g *grafanaInstance) GetGrafanaSession() (string, error) {
	session, err := g.getGrafanaSessionCookie()
	if err != nil {
		return "", err
	}

	return session.Value, nil
}

// wait for the grafana instance to start up
func (g *grafanaInstance) WaitForLiveGrafana(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	if g.isLive() {
		return nil
	}

	timer := time.NewTicker(g.backoff)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if g.isLive() {
				return nil
			}
		case <-ctxTimeout.Done():
			if errors.Is(context.DeadlineExceeded, ctx.Err()) {
				return InstanceNotAvailableError
			}
			return ctx.Err()
		}
	}
}

// checks if grafana is alive
func (g *grafanaInstance) isLive() bool {
	_, err := net.Dial("tcp", g.host)
	return err == nil
}

// getGrafanaSessionCookie returns a session cookie and logs in if none is set
func (g *grafanaInstance) getGrafanaSessionCookie() (*http.Cookie, error) {
	if g.session != nil {
		return g.session, nil
	}

	loginURL := g.Address() + "/login"

	loginPayload := struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}{
		User:     g.user,
		Password: g.password,
	}

	jsonPayload, err := json.Marshal(loginPayload)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	deadline := time.Now().Add(g.timeout)

	for {
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Request failed: %w", err)
		}
		defer resp.Body.Close()

		// check response status code
		responsePayload, _ := io.ReadAll(resp.Body)

		switch resp.StatusCode {
		case http.StatusServiceUnavailable:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf(
					"%w timeout of '%.2fs' exceeded: %s",
					InstanceNotAvailableError,
					g.timeout.Seconds(),
					responsePayload,
				)
			}
			time.Sleep(g.backoff)
			continue

		case http.StatusOK:
			if len(resp.Cookies()) == 0 {
				return nil, fmt.Errorf("no session returned %s", string(responsePayload))
			}

			g.session = resp.Cookies()[0]
			return g.session, nil

		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: login. : %s", InvalidCredentialsError, responsePayload)

		case http.StatusBadRequest:
			if strings.Contains(string(responsePayload), "auth.client.notConfigured") {
				return nil, LoginDisableError
			}

			return nil, fmt.Errorf(
				"%w: login. statusCode: %d, response: %s",
				FailedRequestError,
				resp.StatusCode,
				responsePayload,
			)

		default:
			return nil, fmt.Errorf(
				"%w: login. statusCode: %d, response: %s",
				FailedRequestError,
				resp.StatusCode,
				responsePayload,
			)
		}
	}
}

func (g *grafanaInstance) GetGrafanaBuildVersion() (string, error) {
	grafanaSession, err := g.getGrafanaSessionCookie()
	if err != nil {
		return "", err
	}

	targetURL := g.address + "/api/frontend/settings"
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(grafanaSession)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: get settings: %w", FailedRequestError, bytes.ErrTooLarge)
	}
	defer resp.Body.Close()

	settings := struct {
		BuildInfo struct {
			Version string `json:"version"`
		} `json:"buildInfo"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&settings)
	if err != nil {
		return "", fmt.Errorf("%w: get settings: Failed to decode response: %w", FailedRequestError, err)
	}

	return settings.BuildInfo.Version, nil
}

// parseAddress takes an address and returns its scheme, host and port components
// Examples:
//
//   https://instance:3000 returns (https, instance.net, 3000)
//   http://instance.grafana.net returns (http, instance, 80) (port inferred from scheme)
//   instance:3000 returns an error (cannot infer the schema from port)
//   instance:80 returns (http, instance, 80) (scheme inferred from port)
func parseAddress(address string) (string, string, string, error) {
	u, err := url.Parse(address)
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing grafana address: %w", err)
	}

	host, port, _ := strings.Cut(u.Host, ":")
	scheme := u.Scheme

	// try to infer the scheme from the standard ports
	if scheme == "" {
		switch port {
		case "443":
			scheme = "https"
		case "80":
			scheme = "http"
		default:
			return "", "", "", fmt.Errorf("unknown scheme: address: %s", address)
		}
	}

	// try to infer the port from scheme
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", "", "", fmt.Errorf("unknown scheme: address: %s", address)
		}
	}

	return scheme, host, port, nil
}
