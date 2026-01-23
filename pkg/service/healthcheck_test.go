package service

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitForServiceLive(t *testing.T) {
	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer listener.Close()

	serverURL := "http://" + listener.Addr().String()

	tests := []struct {
		name      string
		url       string
		opts      HealthCheckOptions
		expectErr bool
	}{
		{
			name: "service is available immediately",
			url:  serverURL,
			opts: HealthCheckOptions{
				Timeout: 5 * time.Second,
				Backoff: 100 * time.Millisecond,
			},
			expectErr: false,
		},
		{
			name: "service not available - timeout",
			url:  "http://127.0.0.1:9999", // Non-existent service
			opts: HealthCheckOptions{
				Timeout: 1 * time.Second,
				Backoff: 100 * time.Millisecond,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := WaitForServiceLive(ctx, tt.url, tt.opts)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestWaitForServiceLive_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := HealthCheckOptions{
		Timeout: 5 * time.Second,
		Backoff: 100 * time.Millisecond,
	}

	err := WaitForServiceLive(ctx, "http://127.0.0.1:9999", opts)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestIsServiceLive(t *testing.T) {
	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer listener.Close()

	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{
			name:     "service is live",
			host:     listener.Addr().String(),
			expected: true,
		},
		{
			name:     "service is not live",
			host:     "127.0.0.1:9999",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isServiceLive(tt.host)
			if result != tt.expected {
				t.Errorf("isServiceLive(%s) = %v, expected %v", tt.host, result, tt.expected)
			}
		})
	}
}
