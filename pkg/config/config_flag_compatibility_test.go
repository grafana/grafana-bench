package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/spf13/pflag"
)

// TestRunStageInLogOutput validates that runStage appears in the log output
func TestRunStageInLogOutput(t *testing.T) {
	// Create a SuiteRun with runStage set
	suiteRun := executor.SuiteRun{
		Id:             "test-run-123",
		RunStage:       "ci-stage-test",
		Service:        "grafana",
		TestExecutor:   "k6",
		BenchRevision:  "v1.0.0",
		GrafanaURL:     "http://localhost:3000",
		GrafanaSlug:    "localhost",
		GrafanaVersion: "11.0.0",
	}

	summary := executor.SuiteRunSummary{
		StartTime:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		TotalDuration: 5000 * time.Millisecond,
		TestRuns:      []executor.TestRunSummary{},
		TestsExecuted: 1,
		TestsPassed:   1,
		TestsFailed:   0,
		TestsError:    0,
	}

	// Capture log output
	var logOutput strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create LogReporter and override logger
	logReporter, err := reporter.NewLogReporter("json", []any{"tool", "bench"})
	if err != nil {
		t.Fatalf("Failed to create LogReporter: %v", err)
	}
	logReporter.Log = logger

	// Generate report
	err = logReporter.Report(context.Background(), suiteRun, summary)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	// Parse log output and verify runStage is present
	logLines := strings.Split(strings.TrimSpace(logOutput.String()), "\n")
	found := false

	for _, line := range logLines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var logEntry map[string]any
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue // Skip lines that aren't JSON
		}

		// Check if this is the suiteRun entry (has anyFailures field)
		if _, isSuiteRun := logEntry["anyFailures"]; isSuiteRun {
			if runStageValue, exists := logEntry["runStage"]; exists {
				if runStageValue == "ci-stage-test" {
					found = true
				} else {
					t.Errorf("runStage in log output = %v, want 'ci-stage-test'", runStageValue)
				}
			} else {
				t.Error("runStage field missing from log output")
			}
			break
		}
	}

	if !found {
		t.Error("runStage field not found in log output")
	}
}

// TestFlagCompatibility validates that --run-stage sets RunStage
func TestFlagCompatibility(t *testing.T) {
	tests := []struct {
		name             string
		flagName         string
		flagValue        string
		expectedRunStage string
	}{
		{
			name:             "run-stage flag sets RunStage",
			flagName:         "run-stage",
			flagValue:        "ci-stage-test",
			expectedRunStage: "ci-stage-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config and flag set
			config := &SuiteRunConfig{}
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			
			// Add the flags (this simulates what AddSuiteRunFlags does)
			AddSuiteRunFlags(fs, config)

			// Set the flag value
			err := fs.Set(tt.flagName, tt.flagValue)
			if err != nil {
				t.Fatalf("Failed to set flag %s: %v", tt.flagName, err)
			}

			// Verify the RunStage field was set correctly
			if config.RunStage != tt.expectedRunStage {
				t.Errorf("RunStage = %q, want %q", config.RunStage, tt.expectedRunStage)
			}
		})
	}
}

// TestRunStageIntegrationWithBuildSuiteRun validates the end-to-end integration
func TestRunStageIntegrationWithBuildSuiteRun(t *testing.T) {
	tests := []struct {
		name      string
		runStage  string
	}{
		{
			name:     "integration with ci stage",
			runStage: "ci",
		},
		{
			name:     "integration with local stage",
			runStage: "local",
		},
		{
			name:     "integration with custom stage",
			runStage: "production-deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal config
			config := &BenchConfig{
				SuiteRun: SuiteRunConfig{
					RunStage: tt.runStage,
					Service:  "grafana",
				},
				TestSuite: TestSuiteConfig{
					Name: "test-suite",
				},
				Test: TestConfig{
					Type: "smoke",
				},
				Service: ServiceConfig{
					Version: "11.0.0",
				},
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

			// Build SuiteRun
			suiteRun, err := config.BuildSuiteRun(logger)
			if err != nil {
				t.Fatalf("BuildSuiteRun failed: %v", err)
			}

			// Verify RunStage is set correctly
			if suiteRun.RunStage != tt.runStage {
				t.Errorf("SuiteRun.RunStage = %q, want %q", suiteRun.RunStage, tt.runStage)
			}

			// Verify the new ID format includes stage and suite name
			// Format: {stage}-{suiteName}-{timestamp}
			// Example: ci-test-suite-2026022-140035
			expectedIdPrefix := tt.runStage + "-test-suite-"
			if !strings.HasPrefix(suiteRun.Id, expectedIdPrefix) {
				t.Errorf("SuiteRun.Id = %q, should start with %q", suiteRun.Id, expectedIdPrefix)
			}
		})
	}
}