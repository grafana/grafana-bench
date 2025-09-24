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

func TestLogReporter_Report(t *testing.T) {
	tests := []struct {
		name     string
		suiteRun executor.SuiteRun
		summary  executor.SuiteRunSummary
		expected map[string]any
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
			expected: map[string]any{
				"runId":          "test-run-123",
				"suiteRun":       "test-suite-smoke",
				"suiteName":      "test-suite",
				"suiteRevision":  "abc123",
				"testTrigger":    "local",
				"testExecutor":   "k6",
				"benchRevision":  "v0.6.1",
				"grafanaUrl":     "http://localhost:3000",
				"grafanaSlug":    "localhost",
				"grafanaVersion": "11.0.0",
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
			expected: map[string]any{
				"runId":          "test-run-456",
				"suiteRun":       "empty-fields-test",
				"suiteName":      "",
				"suiteRevision":  "",
				"testTrigger":    "ci",
				"testExecutor":   "playwright",
				"benchRevision":  "dev",
				"grafanaUrl":     "https://grafana.com",
				"grafanaSlug":    "grafana",
				"grafanaVersion": "10.0.0",
			},
		},
		{
			name: "suite run with custom attributes",
			suiteRun: executor.SuiteRun{
				Name:           "attributes-test",
				Id:             "test-run-789",
				Trigger:        "local",
				TestExecutor:   "k6",
				SuiteName:      "smoke-tests",
				SuiteRevision:  "main-branch",
				BenchRevision:  "v0.7.0",
				GrafanaURL:     "http://localhost:3000",
				GrafanaSlug:    "localhost",
				GrafanaVersion: "11.0.0",
			},
			summary: executor.SuiteRunSummary{
				StartTime:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				TotalDuration: 3000 * time.Millisecond,
				TestRuns: []executor.TestRunSummary{
					{
						TestFile:         "smoke.js",
						TestFolder:       "tests",
						Status:           executor.TestPassed,
						ScenarioDuration: 800 * time.Millisecond,
						TotalDuration:    1000 * time.Millisecond,
						Iterations:       "1",
					},
				},
				TestsExecuted: 1,
				TestsPassed:   1,
				TestsFailed:   0,
				TestsError:    0,
				Attributes: map[string]string{
					"environment": "staging",
					"team":        "backend",
					"build_id":    "12345",
					"branch":      "feature/test",
				},
			},
			expected: map[string]any{
				"runId":          "test-run-789",
				"suiteRun":       "attributes-test",
				"suiteName":      "smoke-tests",
				"suiteRevision":  "main-branch",
				"testTrigger":    "local",
				"testExecutor":   "k6",
				"benchRevision":  "v0.7.0",
				"grafanaUrl":     "http://localhost:3000",
				"grafanaSlug":    "localhost",
				"grafanaVersion": "11.0.0",
				"environment":    "staging",
				"team":           "backend",
				"build_id":       "12345",
				"branch":         "feature/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Create LogReporter with JSON format and redirect to buffer
			reporter, err := NewLogReporter("json", []any{"service", "bench"})
			if err != nil {
				t.Fatalf("Failed to create LogReporter: %v", err)
			}

			// Override the logger to write to our buffer instead of stdout
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
			reporter.Log = logger

			err = reporter.Report(context.Background(), tt.suiteRun, tt.summary)
			if err != nil {
				t.Fatalf("Report failed: %v", err)
			}

			// Parse each line of JSON output
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if len(lines) == 0 {
				t.Fatal("No log output generated")
			}

			// Find the suiteRun log entry (contains anyFailures field)
			var suiteRunEntry map[string]any
			for i, line := range lines {
				if line == "" {
					continue
				}

				var logEntry map[string]any
				if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
					t.Fatalf("Failed to parse JSON log entry %d: %v\nLine: %s", i, err, line)
				}

				// Check if this is the suiteRun entry (has anyFailures field)
				if _, isSuiteRun := logEntry["anyFailures"]; isSuiteRun {
					suiteRunEntry = logEntry
				}

				// Verify required fields exist in all entries
				requiredFields := []string{"time", "level", "msg"}
				for _, field := range requiredFields {
					if _, exists := logEntry[field]; !exists {
						t.Errorf("Log entry %d missing required field %q\nLog entry: %s", i, field, line)
					}
				}
			}

			// Verify the suiteRun entry contains all expected fields
			if suiteRunEntry == nil {
				t.Fatal("No suiteRun log entry found (should contain anyFailures field)")
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := suiteRunEntry[key]
				if !exists {
					t.Errorf("SuiteRun log entry missing field %q", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("SuiteRun log entry field %q = %v, want %v",
						key, actualValue, expectedValue)
				}
			}
		})
	}
}

