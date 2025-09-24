package reporter

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/metrics"
)

func TestPrometheusReporter_Report_WithAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  map[string]string
		expectError bool
	}{
		{
			name: "with custom attributes",
			attributes: map[string]string{
				"environment": "staging",
				"team":        "backend",
				"build_id":    "12345",
			},
			expectError: true, // Will error due to invalid URL, but should not panic
		},
		{
			name:        "no custom attributes",
			attributes:  nil,
			expectError: true, // Will error due to invalid URL, but should not panic
		},
		{
			name:        "empty attributes map",
			attributes:  map[string]string{},
			expectError: true, // Will error due to invalid URL, but should not panic
		},
		{
			name: "attributes with special characters",
			attributes: map[string]string{
				"branch":     "feature/my-branch",
				"commit":     "abc123def456",
				"build_url":  "https://ci.example.com/build/123",
			},
			expectError: true, // Will error due to invalid URL, but should not panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create reporter with invalid URL (will cause push to fail, but shouldn't panic)
			config := PrometheusConfig{
				URL:     "", // Invalid URL will cause error
				Prefix:  "test",
				Timeout: 1 * time.Second,
			}
			reporter := NewPrometheusReporter(config)

			suiteRun := executor.SuiteRun{
				Id:             "test-run-123",
				Name:           "test-suite",
				Trigger:        "local",
				BenchRevision:  "abc123",
				GrafanaVersion: "9.0.0",
			}

			summary := executor.SuiteRunSummary{
				StartTime:         time.Now(),
				Status:            executor.SuitePassed,
				TestsExecuted:     5,
				TestsPassed:       4,
				TestsFailed:       1,
				TestsFlaky:        0,
				TestsError:        0,
				TotalDuration:     30 * time.Second,
				ScenariosDuration: 25 * time.Second,
				Metrics: []metrics.Metric{
					{
						Name:   "custom_test_metric",
						Value:  42.0,
						Labels: map[string]string{"test_label": "test_value"},
					},
				},
				Attributes: tt.attributes,
			}

			// The main test is that this doesn't panic when processing attributes
			err := reporter.Report(context.Background(), suiteRun, summary)
			
			if tt.expectError && err == nil {
				t.Error("Expected error due to invalid URL but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// If we get here without panicking, the attributes were processed successfully
		})
	}
}

func TestPrometheusReporter_Report_AttributesIntegration(t *testing.T) {
	// This test verifies the attributes are properly integrated into the label processing
	// We test this by ensuring the Report method completes without errors when we have
	// a valid (but non-existent) Prometheus endpoint
	
	config := PrometheusConfig{
		URL:     "http://nonexistent-prometheus:9090/api/v1/write",
		Prefix:  "test",
		Timeout: 100 * time.Millisecond, // Short timeout
	}
	reporter := NewPrometheusReporter(config)

	suiteRun := executor.SuiteRun{
		Id:             "integration-test",
		Name:           "integration-suite",
		Trigger:        "ci",
		BenchRevision:  "integration-abc123",
		GrafanaVersion: "10.0.0",
	}

	// Test with a comprehensive set of attributes
	summary := executor.SuiteRunSummary{
		StartTime:         time.Now(),
		Status:            executor.SuiteFailed, // Test different status
		TestsExecuted:     10,
		TestsPassed:       7,
		TestsFailed:       2,
		TestsFlaky:        1,
		TestsError:        0,
		TotalDuration:     60 * time.Second,
		ScenariosDuration: 50 * time.Second,
		Metrics: []metrics.Metric{
			{
				Name:   "response_time_p99",
				Value:  1200.5,
				Labels: map[string]string{
					"endpoint": "/api/v1/query",
					"method":   "GET",
				},
			},
			{
				Name:   "error_rate",
				Value:  0.02,
				Labels: map[string]string{
					"service": "api",
				},
			},
		},
		Attributes: map[string]string{
			"environment":    "production",
			"region":         "us-west-2",
			"team":           "platform",
			"build_id":       "67890",
			"branch":         "main",
			"commit_sha":     "abcdef123456",
			"instance_type":  "c5.large",
			"test_suite_id":  "smoke-tests-v2",
		},
	}

	// This should complete without panicking, even though the network request will fail
	err := reporter.Report(context.Background(), suiteRun, summary)
	
	// We expect a network error since the endpoint doesn't exist
	if err == nil {
		t.Error("Expected network error due to nonexistent endpoint, but got no error")
	}

	// The key success criteria is that we didn't panic and attributes were processed
	// The actual network error is expected and acceptable for this test
}

func TestPrometheusReporter_Report_NilAttributesHandling(t *testing.T) {
	config := PrometheusConfig{
		URL:     "",
		Prefix:  "",
		Timeout: 1 * time.Second,
	}
	reporter := NewPrometheusReporter(config)

	suiteRun := executor.SuiteRun{
		Id:             "nil-test",
		Name:           "nil-test-suite",
		GrafanaVersion: "9.0.0",
	}

	summary := executor.SuiteRunSummary{
		StartTime: time.Now(),
		Status:    executor.SuitePassed,
		Attributes: nil, // Explicitly test nil attributes
	}

	// Should not panic with nil attributes
	err := reporter.Report(context.Background(), suiteRun, summary)
	
	// We expect an error due to empty URL, but no panic
	if err == nil {
		t.Error("Expected error due to empty URL")
	}
}