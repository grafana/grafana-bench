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
	"regexp"
	"slices"
	"strings"
	"time"
)

// GrafanaInstance represents an endpoint for accessing a Grafana Instance
type GrafanaInstance interface {
	// Url returns the url to access the grafana instance
	Url() string

	// Hostname returns the grafana instance host name
	Hostname() string

	// Slug returns the grafana instance slug
	Slug() string

	// AdminUser returns the instance's admin user
	AdminUser() string

	// AdminPassword returns the password for the admin user
	AdminPassword() string

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

	slugEx = regexp.MustCompile(`(^|\.)grafana(-dev)?\.net$`)
)

type grafanaInstance struct {
	url           *url.URL
	adminUser     string
	adminPassword string
	session       *http.Cookie
	timeout       time.Duration
	backoff       time.Duration
}

// InstanceOption defines an option for configuring the grafana instance
type InstanceOption func(*grafanaInstance) error

// Sets the grafana timeout. If 0, the default is used
func WithTimeout(timeout time.Duration) InstanceOption {
	return func(g *grafanaInstance) error {
		if timeout != 0 {
			g.timeout = timeout
		}
		return nil
	}
}

// Sets the grafana backoff time. If 0, the default is used
func WithBackoff(backoff time.Duration) InstanceOption {
	return func(g *grafanaInstance) error {
		if backoff != 0 {
			g.backoff = backoff
		}
		return nil
	}
}

func Slug(grafanaURL string) (string, error) {
	u, err := url.Parse(grafanaURL)
	if err != nil {
		return "", fmt.Errorf("invalid grafana url: %w", err)
	}
	grafanaSlug := slugEx.ReplaceAllString(u.Hostname(), "")

	return grafanaSlug, nil
}

// NewGrafanaInstance creates a reference to access a grafana instance
// Takes a fully qualified address such as https://jefflevinslunch.grafana.net
// and a user credentials
func NewInstance(address, adminUser, password string, opts ...InstanceOption) (GrafanaInstance, error) {
	url, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	instance := &grafanaInstance{
		url:           url,
		adminUser:     adminUser,
		adminPassword: password,
		timeout:       DefaultGrafanaTimeout,
		backoff:       DefaultGrafanaBackoff,
	}

	for _, optFunc := range opts {
		if err = optFunc(instance); err != nil {
			return nil, fmt.Errorf("invalid option %v", err)
		}
	}

	return instance, nil
}

// Url returns the url to access the grafana instance
func (g *grafanaInstance) Url() string {
	return g.url.String()
}

// Host returns the grafana instance Hostname
func (g *grafanaInstance) Hostname() string {
	return g.url.Hostname()
}

// Slug returns the grafana instance slug
func (g *grafanaInstance) Slug() string {
	// ignore error, we already validated the url
	slug, _ := Slug(g.url.String())
	return slug
}

// AdminUser returns the instance's admin user
func (g *grafanaInstance) AdminUser() string {
	return g.adminUser
}

// AdminPassword returns the password for the admin user
func (g *grafanaInstance) AdminPassword() string {
	return g.adminPassword
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
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return InstanceNotAvailableError
			}
			return ctx.Err()
		}
	}
}

// checks if grafana is alive
func (g *grafanaInstance) isLive() bool {
	_, err := net.Dial("tcp", g.url.Host)
	return err == nil
}

// getGrafanaSessionCookie returns a session cookie and logs in if none is set
func (g *grafanaInstance) getGrafanaSessionCookie() (*http.Cookie, error) {
	if g.session != nil {
		return g.session, nil
	}

	loginURL := g.Url() + "/login"

	loginPayload := struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}{
		User:     g.adminUser,
		Password: g.adminPassword,
	}

	jsonPayload, err := json.Marshal(loginPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	deadline := time.Now().Add(g.timeout)

	for {
		resp, err := client.Do(req)
		if err != nil {
			// Check if we've exceeded the timeout
			if time.Now().After(deadline) {
				return nil, fmt.Errorf(
					"%w timeout of '%.2fs' exceeded: request failed: %w",
					InstanceNotAvailableError,
					g.timeout.Seconds(),
					err,
				)
			}
			// Retry on connection errors (connection refused, etc.)
			time.Sleep(g.backoff)
			continue
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
			return nil, InvalidCredentialsError

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

	targetURL := g.url.String() + "/api/frontend/settings"
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
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

// parseAddress takes an address and returns its scheme, host, port and path components
// Examples:
//
//	https://instance:3000 returns (https://instance:3000)
//	https://instance:3000/grafana returns (https://instance:3000/grafana)
//	http://instance returns (http://instance:80) (port inferred from scheme)
//	http://instance/grafana returns (http://instance:80/grafana)
func parseAddress(address string) (*url.URL, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("error parsing grafana address: %w", err)
	}
	host, port, _ := strings.Cut(u.Host, ":")
	scheme := u.Scheme
	path := u.Path

	// no scheme
	if scheme == "" {
		return nil, fmt.Errorf("no scheme provided. please add http or https to your grafana url: %s", address)
	}

	// unsupported scheme
	if !slices.Contains([]string{"http", "https"}, scheme) {
		return nil, fmt.Errorf("invalid scheme provided address: %s, scheme: %s", address, scheme)
	}

	// derive port from scheme if not provided
	if scheme == "http" && port == "" {
		port = "80"
	} else if scheme == "https" && port == "" {
		port = "443"
	}

	return url.Parse(fmt.Sprintf("%s://%s:%s%s", scheme, host, port, path))
}
