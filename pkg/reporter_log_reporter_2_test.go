package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func TestLogReporter_Report_WithAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]string
		format     string
	}{
		{
			name: "JSON format with custom attributes",
			attributes: map[string]string{
				"environment": "staging",
				"team":        "backend", 
				"build_id":    "12345",
			},
			format: JSONLog,
		},
		{
			name: "Text format with custom attributes", 
			attributes: map[string]string{
				"env":    "prod",
				"region": "us-west",
			},
			format: TextLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture log output
			var buf bytes.Buffer
			
			// Create a custom logger that writes to our buffer
			var logger *slog.Logger
			switch tt.format {
			case JSONLog:
				logger = slog.New(slog.NewJSONHandler(&buf, nil))
			case TextLog:
				logger = slog.New(slog.NewTextHandler(&buf, nil))
			}
			
			// Add service attribute like the real reporter does
			logger = logger.With("service", "bench")
			
			reporter := &LogReporter{
				Log: logger,
			}

			suiteRun := executor.SuiteRun{
				Id:             "test-run-123",
				Name:           "test-suite",
				Trigger:        "local",
				TestExecutor:   "k6",
				BenchRevision:  "abc123",
				GrafanaURL:     "http://localhost:3000",
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
				TestRuns:          []executor.TestRunSummary{},
				Attributes:        tt.attributes,
			}

			err := reporter.Report(context.Background(), suiteRun, summary)
			if err != nil {
				t.Fatalf("Report() error = %v", err)
			}

			output := buf.String()

			if tt.format == JSONLog {
				// For JSON, parse and verify attributes are present
				lines := strings.Split(strings.TrimSpace(output), "\n")
				var suiteRunLine string
				for _, line := range lines {
					if strings.Contains(line, "suiteRun") && strings.Contains(line, "anyFailures") {
						suiteRunLine = line
						break
					}
				}

				if suiteRunLine == "" {
					t.Fatal("Could not find suiteRun log line in output")
				}

				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(suiteRunLine), &logEntry); err != nil {
					t.Fatalf("Failed to parse JSON log: %v", err)
				}

				// Verify our custom attributes are present
				for key, expectedValue := range tt.attributes {
					if actualValue, ok := logEntry[key]; !ok {
						t.Errorf("Expected attribute %q to be present in log", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected attribute %q to have value %q, got %q", key, expectedValue, actualValue)
					}
				}
			} else {
				// For text format, verify attributes appear in output
				for key, expectedValue := range tt.attributes {
					expectedPattern := key + "=" + expectedValue
					if !strings.Contains(output, expectedPattern) {
						t.Errorf("Expected text log to contain %q, but it didn't.\nFull output:\n%s", expectedPattern, output)
					}
				}
			}
		})
	}
}

func TestLogReporter_Report_NoAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	reporter := &LogReporter{Log: logger}

	suiteRun := executor.SuiteRun{Id: "test", Name: "test"}
	summary := executor.SuiteRunSummary{
		Status:     executor.SuitePassed,
		Attributes: nil,
	}

	err := reporter.Report(context.Background(), suiteRun, summary)
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	// Should not error with nil attributes
}

