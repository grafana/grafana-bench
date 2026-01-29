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
				Id:             "test-run-123",
				RunStage:       "local",
				Service:        "grafana",
				TestExecutor:   "k6",
				BenchRevision:  "v0.6.1",
				ServiceURL:     "http://localhost:3000",
				ServiceVersion: "11.0.0",
			},
			summary: executor.SuiteRunSummary{
				SuiteName:      "test-suite",
				SuiteRevision:  "abc123",
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
				"suiteName":      "test-suite",
				"suiteRevision":  "abc123",
				"runStage":       "local",
				"testExecutor":   "k6",
				"benchRevision":  "v0.6.1",
				"serviceUrl":     "http://localhost:3000",
				"serviceVersion": "11.0.0",
			},
		},
		{
			name: "suite run with empty optional fields",
			suiteRun: executor.SuiteRun{
				Id:             "test-run-456",
				RunStage:       "ci",
				Service:        "grafana",
				TestExecutor:   "playwright",
				BenchRevision:  "dev",
				ServiceURL:     "https://grafana.com",
				ServiceVersion: "10.0.0",
			},
			summary: executor.SuiteRunSummary{
				SuiteName:      "",
				SuiteRevision:  "",
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
				"suiteName":      "",
				"suiteRevision":  "",
				"runStage":       "ci",
				"testExecutor":   "playwright",
				"benchRevision":  "dev",
				"serviceUrl":     "https://grafana.com",
				"serviceVersion": "10.0.0",
			},
		},
		{
			name: "suite run with custom attributes",
			suiteRun: executor.SuiteRun{
				Id:             "test-run-789",
				RunStage:       "local",
				Service:        "grafana",
				TestExecutor:   "k6",
				BenchRevision:  "v0.7.0",
				ServiceURL:     "http://localhost:3000",
				ServiceVersion: "11.0.0",
				Attributes: map[string]string{
					"environment": "staging",
					"team":        "backend",
					"build_id":    "12345",
					"branch":      "feature/test",
				},
			},
			summary: executor.SuiteRunSummary{
				SuiteName:      "smoke-tests",
				SuiteRevision:  "main-branch",
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
			},
			expected: map[string]any{
				"runId":          "test-run-789",
				"suiteName":      "smoke-tests",
				"suiteRevision":  "main-branch",
				"runStage":       "local",
				"testExecutor":   "k6",
				"benchRevision":  "v0.7.0",
				"serviceUrl":     "http://localhost:3000",
				"serviceVersion": "11.0.0",
				"environment":    "staging",
				"team":           "backend",
				"build_id":       "12345",
				"branch":         "feature/test",
			},
		},
		{
			name: "suite run for gobench without serviceUrl",
			suiteRun: executor.SuiteRun{
				Id:             "test-run-999",
				RunStage:       "local",
				Service:        "bench",
				TestExecutor:   "gobench",
				BenchRevision:  "v1.0.0",
				ServiceURL:     "", // Empty for go/gobench
				ServiceVersion: "main-abc123",
			},
			summary: executor.SuiteRunSummary{
				SuiteName:      "benchmarks",
				SuiteRevision:  "main",
				StartTime:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				TotalDuration:  2000 * time.Millisecond,
				TestRuns:       []executor.TestRunSummary{},
				TestsExecuted:  2,
				TestsPassed:    2,
				TestsFailed:    0,
				TestsError:     0,
			},
			expected: map[string]any{
				"runId":          "test-run-999",
				"suiteName":      "benchmarks",
				"suiteRevision":  "main",
				"runStage":       "local",
				"testExecutor":   "gobench",
				"benchRevision":  "v1.0.0",
				// Note: serviceUrl should NOT be present in the log
				"serviceVersion": "main-abc123",
			},
		},
		{
			name: "suite run for gotest without serviceUrl",
			suiteRun: executor.SuiteRun{
				Id:             "test-run-888",
				RunStage:       "ci",
				Service:        "bench",
				TestExecutor:   "go",
				BenchRevision:  "v1.0.0",
				ServiceURL:     "", // Empty for go/gobench
				ServiceVersion: "main-def456",
			},
			summary: executor.SuiteRunSummary{
				SuiteName:      "unit-tests",
				SuiteRevision:  "main",
				StartTime:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				TotalDuration:  1500 * time.Millisecond,
				TestRuns:       []executor.TestRunSummary{},
				TestsExecuted:  5,
				TestsPassed:    5,
				TestsFailed:    0,
				TestsError:     0,
			},
			expected: map[string]any{
				"runId":          "test-run-888",
				"suiteName":      "unit-tests",
				"suiteRevision":  "main",
				"runStage":       "ci",
				"testExecutor":   "go",
				"benchRevision":  "v1.0.0",
				// Note: serviceUrl should NOT be present in the log
				"serviceVersion": "main-def456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Create LogReporter with JSON format and redirect to buffer
			reporter, err := NewLogReporter("json", []any{"tool", "bench"})
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

			// For gobench/gotest executors, verify serviceUrl is NOT present
			if tt.suiteRun.TestExecutor == "gobench" || tt.suiteRun.TestExecutor == "go" {
				if _, hasServiceUrl := suiteRunEntry["serviceUrl"]; hasServiceUrl {
					t.Errorf("SuiteRun log entry should not contain 'serviceUrl' for executor %q", tt.suiteRun.TestExecutor)
				}
			}
		})
	}
}
