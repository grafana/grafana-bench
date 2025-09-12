package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func TestLogReporter_Report(t *testing.T) {
	tests := []struct {
		name     string
		suiteRun executor.SuiteRun
		summary  executor.SuiteRunSummary
		expected map[string]interface{}
	}{
		{
			name: "complete suite run with all fields",
			suiteRun: executor.SuiteRun{
				Name:           "test-suite-smoke",
				Id:             "test-run-123",
				Trigger:        "local",
				TestExecutor:   "k6",
				SuiteName:      "test-suite",
				SuiteRevision:  "abc123",
				BenchRevision:  "v0.6.1",
				GrafanaURL:     "http://localhost:3000",
				GrafanaSlug:    "localhost",
				GrafanaVersion: "11.0.0",
			},
			summary: executor.SuiteRunSummary{
				StartTime:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				TotalDuration: 5000 * time.Millisecond,
				TestRuns: []executor.TestRunSummary{
					{
						TestFile:         "test1.js",
						TestFolder:       "tests",
						Status:           executor.TestPassed,
						ScenarioDuration: 1000 * time.Millisecond,
						TotalDuration:    1200 * time.Millisecond,
						Iterations:       "1",
					},
				},
				TestsExecuted: 1,
				TestsPassed:   1,
				TestsFailed:   0,
				TestsError:    0,
			},
			expected: map[string]interface{}{
				"runId":           "test-run-123",
				"suiteRun":        "test-suite-smoke",
				"suiteName":       "test-suite",
				"suiteRevision":   "abc123",
				"testTrigger":     "local",
				"testExecutor":    "k6",
				"benchRevision":   "v0.6.1",
				"grafanaUrl":      "http://localhost:3000",
				"grafanaSlug":     "localhost",
				"grafanaVersion":  "11.0.0",
			},
		},
		{
			name: "suite run with empty optional fields",
			suiteRun: executor.SuiteRun{
				Name:           "empty-fields-test",
				Id:             "test-run-456",
				Trigger:        "ci",
				TestExecutor:   "playwright",
				SuiteName:      "",
				SuiteRevision:  "",
				BenchRevision:  "dev",
				GrafanaURL:     "https://grafana.com",
				GrafanaSlug:    "grafana",
				GrafanaVersion: "10.0.0",
			},
			summary: executor.SuiteRunSummary{
				StartTime:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				TotalDuration: 2000 * time.Millisecond,
				TestRuns:      []executor.TestRunSummary{},
				TestsExecuted: 0,
				TestsPassed:   0,
				TestsFailed:   0,
				TestsError:    0,
			},
			expected: map[string]interface{}{
				"runId":           "test-run-456",
				"suiteRun":        "empty-fields-test",
				"suiteName":       "",
				"suiteRevision":   "",
				"testTrigger":     "ci",
				"testExecutor":    "playwright",
				"benchRevision":   "dev",
				"grafanaUrl":      "https://grafana.com",
				"grafanaSlug":     "grafana",
				"grafanaVersion":  "10.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			
			// Create logger that writes to buffer with both service and serviceName attributes
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})).With("service", "bench", "serviceName", "test-service")
			
			// Create LogReporter directly with our custom logger
			reporter := &LogReporter{Log: logger}

			err := reporter.Report(context.Background(), tt.suiteRun, tt.summary)
			if err != nil {
				t.Fatalf("Report failed: %v", err)
			}

			// Parse each line of JSON output
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(lines) == 0 {
				t.Fatal("No log output generated")
			}

			// Check that all expected fields are present in all log entries
			for i, line := range lines {
				if line == "" {
					continue
				}

				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
					t.Fatalf("Failed to parse JSON log entry %d: %v\nLine: %s", i, err, line)
				}

				// Verify all expected fields are present
				for key, expectedValue := range tt.expected {
					actualValue, exists := logEntry[key]
					if !exists {
						t.Errorf("Log entry %d missing field %q\nLog entry: %s", i, key, line)
						continue
					}

					if actualValue != expectedValue {
						t.Errorf("Log entry %d field %q = %v, want %v\nLog entry: %s", 
							i, key, actualValue, expectedValue, line)
					}
				}

				// Verify required fields exist
				requiredFields := []string{"time", "level", "msg"}
				for _, field := range requiredFields {
					if _, exists := logEntry[field]; !exists {
						t.Errorf("Log entry %d missing required field %q\nLog entry: %s", i, field, line)
					}
				}
			}
		})
	}
}

func TestLogReporter_SuiteNameFieldPresent(t *testing.T) {
	// This specific test ensures the suiteName field is always present
	var buf bytes.Buffer
	
	// Create logger that writes to buffer with both service and serviceName attributes
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With("service", "bench", "serviceName", "my-test-service")
	
	// Create LogReporter directly with our custom logger
	reporter := &LogReporter{Log: logger}

	suiteRun := executor.SuiteRun{
		Name:           "test-suite-smoke",
		Id:             "test-123",
		Trigger:        "local",
		TestExecutor:   "k6",
		SuiteName:      "my-test-suite",
		SuiteRevision:  "commit-abc",
		BenchRevision:  "v0.6.1",
		GrafanaURL:     "http://localhost:3000",
		GrafanaSlug:    "localhost",
		GrafanaVersion: "11.0.0",
	}

	summary := executor.SuiteRunSummary{
		StartTime:     time.Now(),
		TotalDuration: 1000 * time.Millisecond,
		TestRuns: []executor.TestRunSummary{
			{
				TestFile:         "sample.js",
				Status:           executor.TestPassed,
				ScenarioDuration: 500 * time.Millisecond,
				TotalDuration:    600 * time.Millisecond,
				Iterations:       "1",
			},
		},
		TestsExecuted: 1,
		TestsPassed:   1,
	}

	err := reporter.Report(context.Background(), suiteRun, summary)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("No log output generated")
	}

	// Check each line contains suiteName
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}

		if !strings.Contains(line, `"suiteName":"my-test-suite"`) {
			t.Errorf("Log line %d does not contain suiteName field:\n%s", i, line)
		}

		if !strings.Contains(line, `"suiteRevision":"commit-abc"`) {
			t.Errorf("Log line %d does not contain suiteRevision field:\n%s", i, line)
		}

		if !strings.Contains(line, `"service":"bench"`) {
			t.Errorf("Log line %d does not contain service field:\n%s", i, line)
		}

		if !strings.Contains(line, `"serviceName":"my-test-service"`) {
			t.Errorf("Log line %d does not contain serviceName field:\n%s", i, line)
		}
	}
}

func TestLogReporter_CustomServiceName(t *testing.T) {
	// Test that custom service names are used correctly
	var buf bytes.Buffer
	
	customServiceName := "my-custom-app"
	
	// Create logger that writes to buffer with both service and serviceName attributes
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With("service", "bench", "serviceName", customServiceName)
	
	// Create LogReporter directly with our custom logger
	reporter := &LogReporter{Log: logger}

	suiteRun := executor.SuiteRun{
		Name:           "test-run",
		Id:             "test-123",
		Trigger:        "local",
		TestExecutor:   "k6",
		SuiteName:      "my-suite",
		SuiteRevision:  "abc123",
		BenchRevision:  "v0.6.1",
		GrafanaURL:     "http://localhost:3000",
		GrafanaSlug:    "localhost",
		GrafanaVersion: "11.0.0",
	}

	summary := executor.SuiteRunSummary{
		StartTime:     time.Now(),
		TotalDuration: 1000 * time.Millisecond,
		TestRuns:      []executor.TestRunSummary{},
		TestsExecuted: 0,
		TestsPassed:   0,
	}

	err := reporter.Report(context.Background(), suiteRun, summary)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("No log output generated")
	}

	// Verify the custom service name appears in all log lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}

		// Check for backward-compatible service="bench" field
		if !strings.Contains(line, `"service":"bench"`) {
			t.Errorf("Log line %d does not contain service field:\n%s", i, line)
		}

		// Check for new serviceName field with custom value
		expectedServiceNameField := fmt.Sprintf(`"serviceName":"%s"`, customServiceName)
		if !strings.Contains(line, expectedServiceNameField) {
			t.Errorf("Log line %d does not contain expected serviceName %q:\n%s", i, customServiceName, line)
		}
	}
}