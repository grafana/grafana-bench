package config

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestSuiteNameRequired verifies that --suite-name is required
func TestSuiteNameRequired(t *testing.T) {
	config := &BenchConfig{
		TestSuite: TestSuiteConfig{
			Name: "", // Empty name should fail
			Path: "/some/path",
		},
		Service: ServiceConfig{
			Name:    "test-service",
			Version: "1.0.0",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	_, err := config.BuildTestSuite(logger)
	if err == nil {
		t.Error("Expected error when suite-name is empty, but got nil")
	}
	if !strings.Contains(err.Error(), "--suite-name is required") {
		t.Errorf("Expected error message to contain '--suite-name is required', got: %v", err)
	}
	// Verify the error message includes helpful examples
	if !strings.Contains(err.Error(), "Examples:") {
		t.Errorf("Expected error message to contain examples, got: %v", err)
	}
}

// TestPrometheusValidation verifies that prometheus flags are validated when --prometheus-metrics is enabled
func TestPrometheusValidation(t *testing.T) {
	tests := []struct {
		name          string
		prometheus    Prometheus
		expectError   bool
		errorContains string
	}{
		{
			name: "all prometheus fields provided",
			prometheus: Prometheus{
				Metrics:  true,
				URL:      "http://localhost:9090",
				User:     "user",
				Password: "pass",
			},
			expectError: false,
		},
		{
			name: "missing URL",
			prometheus: Prometheus{
				Metrics:  true,
				URL:      "",
				User:     "user",
				Password: "pass",
			},
			expectError:   true,
			errorContains: "PROMETHEUS_URL",
		},
		{
			name: "missing User",
			prometheus: Prometheus{
				Metrics:  true,
				URL:      "http://localhost:9090",
				User:     "",
				Password: "pass",
			},
			expectError:   true,
			errorContains: "PROMETHEUS_USER",
		},
		{
			name: "missing Password",
			prometheus: Prometheus{
				Metrics:  true,
				URL:      "http://localhost:9090",
				User:     "user",
				Password: "",
			},
			expectError:   true,
			errorContains: "PROMETHEUS_PASSWORD",
		},
		{
			name: "metrics disabled - no validation needed",
			prometheus: Prometheus{
				Metrics:  false,
				URL:      "",
				User:     "",
				Password: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BenchConfig{
				TestSuite: TestSuiteConfig{
					Name: "test-suite",
					Path: "/some/path",
				},
				Service: ServiceConfig{
					Name:    "test-service",
					Version: "1.0.0",
				},
				Report: ReportConfig{
					Output: "log",
				},
				Prometheus: tt.prometheus,
			}

			_, err := config.BuildReporter()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectError && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
			}
		})
	}
}
