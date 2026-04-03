package service

import (
	"context"
	"errors"
	"net"
	"net/url"
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
}

// DefaultHealthCheckOptions returns the default health check configuration
func DefaultHealthCheckOptions() HealthCheckOptions {
	return HealthCheckOptions{
		Timeout: 60 * time.Second,
		Backoff: 1 * time.Second,
	}
}

// WaitForServiceLive performs a TCP health check on the given service URL
// It repeatedly dials the service until it's available or the timeout is reached
func WaitForServiceLive(ctx context.Context, serviceURL string, opts HealthCheckOptions) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}

	host := parsedURL.Host
	if parsedURL.Port() == "" {
		switch parsedURL.Scheme {
		case "https":
			host = parsedURL.Hostname() + ":443"
		case "http":
			host = parsedURL.Hostname() + ":80"
		}
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Check if already live
	if isServiceLive(host) {
		return nil
	}

	ticker := time.NewTicker(opts.Backoff)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if isServiceLive(host) {
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

// isServiceLive checks if the service is available by attempting a TCP dial
func isServiceLive(host string) bool {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
