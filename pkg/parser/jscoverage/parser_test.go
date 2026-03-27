package jscoverage

import (
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func TestParseCoverageJSON(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		codeowner       string
		pkg             string
		expectedStatus  executor.SuiteStatus
		expectedMetrics int
		validateMetrics func(t *testing.T, summary executor.SuiteRunSummary)
	}{
		{
			name:            "full coverage - all metrics present",
			filename:        "testdata/full-coverage.json",
			codeowner:       "@grafana/datapro",
			pkg:             "@grafana/grafana",
			expectedStatus:  executor.SuitePassed,
			expectedMetrics: 4,
			validateMetrics: func(t *testing.T, summary executor.SuiteRunSummary) {
				metricMap := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricMap[m.Name] = m.Value

					if m.Labels["codeowner"] != "@grafana/datapro" {
						t.Errorf("Expected codeowner label '@grafana/datapro', got '%s'", m.Labels["codeowner"])
					}
					if m.Labels["package"] != "@grafana/grafana" {
						t.Errorf("Expected package label '@grafana/grafana', got '%s'", m.Labels["package"])
					}
				}

				if val, ok := metricMap["js_coverage_lines_percent"]; !ok || val != 84.77 {
					t.Errorf("Expected lines coverage 84.77, got %v", val)
				}
				if val, ok := metricMap["js_coverage_statements_percent"]; !ok || val != 85.3 {
					t.Errorf("Expected statements coverage 85.3, got %v", val)
				}
				if val, ok := metricMap["js_coverage_functions_percent"]; !ok || val != 89.1 {
					t.Errorf("Expected functions coverage 89.1, got %v", val)
				}
				if val, ok := metricMap["js_coverage_branches_percent"]; !ok || val != 81.29 {
					t.Errorf("Expected branches coverage 81.29, got %v", val)
				}
			},
		},
		{
			name:            "partial coverage - branches missing",
			filename:        "testdata/partial-coverage.json",
			codeowner:       "@grafana/frontend",
			pkg:             "@grafana/ui",
			expectedStatus:  executor.SuitePassed,
			expectedMetrics: 3,
			validateMetrics: func(t *testing.T, summary executor.SuiteRunSummary) {
				metricMap := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricMap[m.Name] = m.Value
				}

				if val, ok := metricMap["js_coverage_lines_percent"]; !ok || val != 44.5 {
					t.Errorf("Expected lines coverage 44.5, got %v", val)
				}
				if val, ok := metricMap["js_coverage_statements_percent"]; !ok || val != 62.56 {
					t.Errorf("Expected statements coverage 62.56, got %v", val)
				}
				if val, ok := metricMap["js_coverage_functions_percent"]; !ok || val != 49.67 {
					t.Errorf("Expected functions coverage 49.67, got %v", val)
				}

				if _, ok := metricMap["js_coverage_branches_percent"]; ok {
					t.Error("Expected branches coverage to be missing")
				}
			},
		},
		{
			name:            "empty coverage - no metrics",
			filename:        "testdata/empty-coverage.json",
			codeowner:       "@grafana/test",
			pkg:             "@grafana/test-pkg",
			expectedStatus:  executor.SuitePassed,
			expectedMetrics: 0,
			validateMetrics: func(t *testing.T, summary executor.SuiteRunSummary) {
				if len(summary.Metrics) != 0 {
					t.Errorf("Expected 0 metrics, got %d", len(summary.Metrics))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tt.filename)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			summary, err := ParseCoverageJSON(file, tt.codeowner, tt.pkg)
			if err != nil {
				t.Fatalf("ParseCoverageJSON failed: %v", err)
			}

			if summary.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, summary.Status)
			}

			if len(summary.Metrics) != tt.expectedMetrics {
				t.Errorf("Expected %d metrics, got %d", tt.expectedMetrics, len(summary.Metrics))
			}

			if summary.Attributes["codeowner"] != tt.codeowner {
				t.Errorf("Expected codeowner attribute '%s', got '%s'", tt.codeowner, summary.Attributes["codeowner"])
			}
			if summary.Attributes["package"] != tt.pkg {
				t.Errorf("Expected package attribute '%s', got '%s'", tt.pkg, summary.Attributes["package"])
			}

			if tt.validateMetrics != nil {
				tt.validateMetrics(t, summary)
			}
		})
	}
}

func TestParseCoverageJSON_InvalidJSON(t *testing.T) {
	r := strings.NewReader("{bad json}")
	_, err := ParseCoverageJSON(r, "@grafana/test", "@grafana/test-pkg")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
