package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ServiceNotAvailableError is returned when the service is not available after the timeout
	ServiceNotAvailableError = errors.New("service not available")
)

// HealthCheckOptions configures the health check behavior
type HealthCheckOptions struct {
	Timeout time.Duration
	Backoff time.Duration
	// Path, when set, turns the check into an HTTP GET of serviceURL+Path that
	// must answer 2xx. A gateway that accepts connections but serves a loading
	// page (503) passes the TCP dial and fails this one. Empty keeps the TCP dial.
	Path string
}

// DefaultHealthCheckOptions returns the default health check configuration
func DefaultHealthCheckOptions() HealthCheckOptions {
	return HealthCheckOptions{
		Timeout: 60 * time.Second,
		Backoff: 1 * time.Second,
	}
}

// httpProbeTimeout bounds one GET of the health path. The overall wait is opts.Timeout.
const httpProbeTimeout = 10 * time.Second

// WaitForServiceLive performs a health check on the given service URL. It repeatedly
// dials the service, or GETs opts.Path on it when set, until it is available or the
// timeout is reached.
func WaitForServiceLive(ctx context.Context, serviceURL string, opts HealthCheckOptions) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	probe := func() bool { return isServiceLive(hostWithPort(parsedURL)) }
	if opts.Path != "" {
		healthURL := strings.TrimRight(serviceURL, "/") + "/" + strings.TrimLeft(opts.Path, "/")
		probe = func() bool { return isServiceHealthy(ctxTimeout, healthURL) }
	}

	// Check if already live
	if probe() {
		return nil
	}

	ticker := time.NewTicker(opts.Backoff)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if probe() {
				return nil
			}
		case <-ctxTimeout.Done():
			if errors.Is(ctxTimeout.Err(), context.DeadlineExceeded) {
				return ServiceNotAvailableError
			}
			return ctxTimeout.Err()
		}
	}
}

// hostWithPort returns the host to dial, adding the scheme's default port when the URL has none
func hostWithPort(parsedURL *url.URL) string {
	host := parsedURL.Host
	if parsedURL.Port() == "" {
		switch parsedURL.Scheme {
		case "https":
			host = parsedURL.Hostname() + ":443"
		case "http":
			host = parsedURL.Hostname() + ":80"
		}
	}
	return host
}

// isServiceHealthy GETs healthURL and reports whether it answered 2xx
func isServiceHealthy(ctx context.Context, healthURL string) bool {
	ctxProbe, cancel := context.WithTimeout(ctx, httpProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxProbe, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// isServiceLive checks if the service is available by attempting a TCP dial
func isServiceLive(host string) bool {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
