package zizmor

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

func TestParseSARIF(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name       string
		testdata   string
		expectErr  error
		checkExtra func(*testing.T, executor.SuiteRunSummary)
	}{
		{
			name:      "SARIF with findings",
			testdata:  "./testdata/with-findings.sarif",
			expectErr: nil,
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				// Check suite-level attributes
				if summary.Attributes == nil {
					t.Fatal("Attributes should not be nil")
				}

				assert.Equal(t, "tool version", "1.14.2", summary.Attributes["tool_version"])
				assert.Equal(t, "total vulnerabilities", "4", summary.Attributes["total_vulnerabilities"])
				assert.Equal(t, "high severity", "2", summary.Attributes["high_severity"])
				assert.Equal(t, "medium severity", "2", summary.Attributes["medium_severity"])
				assert.Equal(t, "low severity", "0", summary.Attributes["low_severity"])
				assert.Equal(t, "unique rules", "3", summary.Attributes["unique_rules_violated"])

				// Check suite status and counts
				assert.Equal(t, "suite name", "zizmor", summary.SuiteName)
				assert.Equal(t, "suite status", executor.SuiteFailed, summary.Status)
				assert.Equal(t, "tests executed", int32(4), summary.TestsExecuted)
				assert.Equal(t, "tests failed", int32(4), summary.TestsFailed)
				assert.Equal(t, "tests passed", int32(0), summary.TestsPassed)
				assert.Equal(t, "tests error", int32(0), summary.TestsError)

				// Check Prometheus metrics are created
				if len(summary.Metrics) == 0 {
					t.Fatal("Expected metrics to be created, got none")
				}

				// Verify core metrics exist
				metricNames := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricNames[m.Name] = m.Value
				}

				// Check expected metrics
				if val, ok := metricNames["bench_zizmor_total_vulnerabilities"]; !ok {
					t.Error("Missing bench_zizmor_total_vulnerabilities metric")
				} else {
					assert.Equal(t, "total vulnerabilities metric", 4.0, val)
				}

				if val, ok := metricNames["bench_zizmor_high_severity"]; !ok {
					t.Error("Missing bench_zizmor_high_severity metric")
				} else {
					assert.Equal(t, "high severity metric", 2.0, val)
				}

				if val, ok := metricNames["bench_zizmor_medium_severity"]; !ok {
					t.Error("Missing bench_zizmor_medium_severity metric")
				} else {
					assert.Equal(t, "medium severity metric", 2.0, val)
				}

				if val, ok := metricNames["bench_zizmor_low_severity"]; !ok {
					t.Error("Missing bench_zizmor_low_severity metric")
				} else {
					assert.Equal(t, "low severity metric", 0.0, val)
				}

				if val, ok := metricNames["bench_zizmor_unique_rules_violated"]; !ok {
					t.Error("Missing bench_zizmor_unique_rules_violated metric")
				} else {
					assert.Equal(t, "unique rules metric", 3.0, val)
				}

				// Check per-rule metrics exist (should have 3 rule violation metrics)
				ruleViolationCount := 0
				for _, m := range summary.Metrics {
					if m.Name == "bench_zizmor_rule_violations" {
						ruleViolationCount++
						// Each rule violation should have value 1.0
						assert.Equal(t, "rule violation value", 1.0, m.Value)
						// Should have rule_id label
						if _, ok := m.Labels["rule_id"]; !ok {
							t.Error("Rule violation metric missing rule_id label")
						}
					}
				}
				assert.Equal(t, "rule violation metrics count", 3, ruleViolationCount)

				// Check individual test runs
				if len(summary.TestRuns) != 4 {
					t.Fatalf("Expected 4 test runs, got %d", len(summary.TestRuns))
				}

				// First result: artipacked (error)
				firstTest := summary.TestRuns[0]
				assert.Equal(t, "first test file", ".github/workflows/ci.yml", firstTest.TestFile)
				assert.Equal(t, "first test status", executor.TestFailed, firstTest.Status)
				assert.Equal(t, "first test rule", "artipacked", firstTest.Attributes["ruleId"])
				assert.Equal(t, "first test severity", "error", firstTest.Attributes["severity"])
				if !strings.Contains(firstTest.ExitMessage, "line 42") {
					t.Errorf("Expected exit message to contain 'line 42', got %q", firstTest.ExitMessage)
				}

				// Second result: dangerous-triggers (warning)
				secondTest := summary.TestRuns[1]
				assert.Equal(t, "second test file", ".github/workflows/pr-check.yml", secondTest.TestFile)
				assert.Equal(t, "second test status", executor.TestFailed, secondTest.Status)
				assert.Equal(t, "second test rule", "dangerous-triggers", secondTest.Attributes["ruleId"])
				assert.Equal(t, "second test severity", "warning", secondTest.Attributes["severity"])
			},
		},
		{
			name:      "SARIF with no findings",
			testdata:  "./testdata/no-findings.sarif",
			expectErr: nil,
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				// Check suite-level attributes
				if summary.Attributes == nil {
					t.Fatal("Attributes should not be nil")
				}

				assert.Equal(t, "tool version", "1.14.2", summary.Attributes["tool_version"])
				assert.Equal(t, "total vulnerabilities", "0", summary.Attributes["total_vulnerabilities"])
				assert.Equal(t, "high severity", "0", summary.Attributes["high_severity"])
				assert.Equal(t, "medium severity", "0", summary.Attributes["medium_severity"])
				assert.Equal(t, "low severity", "0", summary.Attributes["low_severity"])

				// Check suite status and counts
				assert.Equal(t, "suite name", "zizmor", summary.SuiteName)
				assert.Equal(t, "suite status", executor.SuitePassed, summary.Status)
				assert.Equal(t, "tests executed", int32(0), summary.TestsExecuted)
				assert.Equal(t, "tests failed", int32(0), summary.TestsFailed)
				assert.Equal(t, "tests passed", int32(0), summary.TestsPassed)
				assert.Equal(t, "tests error", int32(0), summary.TestsError)

				// Check metrics are still created (with zero values)
				if len(summary.Metrics) == 0 {
					t.Fatal("Expected metrics to be created even with no findings")
				}

				// Verify zero-value metrics
				metricNames := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricNames[m.Name] = m.Value
				}

				assert.Equal(t, "total vulnerabilities metric", 0.0, metricNames["bench_zizmor_total_vulnerabilities"])
				assert.Equal(t, "high severity metric", 0.0, metricNames["bench_zizmor_high_severity"])
				assert.Equal(t, "medium severity metric", 0.0, metricNames["bench_zizmor_medium_severity"])
				assert.Equal(t, "low severity metric", 0.0, metricNames["bench_zizmor_low_severity"])
				assert.Equal(t, "unique rules metric", 0.0, metricNames["bench_zizmor_unique_rules_violated"])

				// No rule violation metrics when there are no findings
				for _, m := range summary.Metrics {
					if m.Name == "bench_zizmor_rule_violations" {
						t.Error("Should not have rule violation metrics when there are no findings")
					}
				}

				// No test runs
				if len(summary.TestRuns) != 0 {
					t.Fatalf("Expected 0 test runs, got %d", len(summary.TestRuns))
				}
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report, err := os.ReadFile(tc.testdata)
			if err != nil {
				t.Fatalf("Could not read test data file: %v", err.Error())
			}

			summary, err := ParseSARIF(bytes.NewBuffer(report))
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("Expected error %v, got %v", tc.expectErr, err)
			}

			if tc.checkExtra != nil {
				tc.checkExtra(t, summary)
			}
		})
	}
}

func TestParseSARIF_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Test with invalid JSON
	invalidJSON := `{"invalid json`
	_, err := ParseSARIF(bytes.NewBufferString(invalidJSON))

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestParseSARIF_NoRuns(t *testing.T) {
	t.Parallel()

	// Test with SARIF that has no runs
	noRuns := `{"version": "2.1.0", "runs": []}`
	_, err := ParseSARIF(bytes.NewBufferString(noRuns))

	if err == nil {
		t.Fatal("Expected error for SARIF with no runs, got nil")
	}

	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("Expected ErrInvalidFormat, got %v", err)
	}
}
