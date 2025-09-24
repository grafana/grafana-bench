package reporter

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func TestTextReporter_Report(t *testing.T) {
	tests := []struct {
		name             string
		summary          executor.SuiteRunSummary
		expectedContains []string
	}{
		{
			name: "basic summary output",
			summary: executor.SuiteRunSummary{
				StartTime:         time.Now(),
				Status:            executor.SuitePassed,
				TestsExecuted:     10,
				TestsPassed:       8,
				TestsFailed:       2,
				TestsFlaky:        1,
				TestsError:        0,
				TotalDuration:     30 * time.Second,
				ScenariosDuration: 25 * time.Second,
				TestRuns: []executor.TestRunSummary{
					{
						TestFile:         "test1.js",
						TestFolder:       "smoke",
						Status:           executor.TestPassed,
						TotalDuration:    10 * time.Second,
						ScenarioDuration: 8 * time.Second,
					},
					{
						TestFile:         "test2.js", 
						TestFolder:       "load",
						Status:           executor.TestFailed,
						TotalDuration:    15 * time.Second,
						ScenarioDuration: 12 * time.Second,
					},
				},
			},
			expectedContains: []string{
				"----------------SUMMARY----------------",
				"Executed:       10",
				"Passed:         8", 
				"Failed:         2",
				"Flaky:          1",
				"Errors:         0",
				"Suite:          passed",
				"Total Run Time: 30.00 sec",
				"[PASSED]",
				"[FAILED]",
				"smoke: test1.js",
				"load:  test2.js",
			},
		},
		{
			name: "with custom attributes",
			summary: executor.SuiteRunSummary{
				Status:        executor.SuiteFailed,
				TestsExecuted: 5,
				TestsPassed:   3,
				TestsFailed:   2,
				TotalDuration: 20 * time.Second,
				TestRuns:      []executor.TestRunSummary{},
				Attributes: map[string]string{
					"environment": "staging",
					"team":        "backend",
					"build_id":    "12345",
				},
			},
			expectedContains: []string{
				"----------------SUMMARY----------------",
				"Executed:       5",
				"Passed:         3",
				"Failed:         2", 
				"Suite:          failed",
				"--------------ATTRIBUTES---------------",
				"environment: staging",
				"team:        backend",
				"build_id:    12345",
			},
		},
		{
			name: "no attributes section when empty",
			summary: executor.SuiteRunSummary{
				Status:        executor.SuitePassed,
				TestsExecuted: 1,
				TestsPassed:   1,
				TotalDuration: 5 * time.Second,
				TestRuns:      []executor.TestRunSummary{},
				Attributes:    map[string]string{},
			},
			expectedContains: []string{
				"----------------SUMMARY----------------",
				"Executed:       1",
				"Passed:         1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewTextReporter(&buf)

			suiteRun := executor.SuiteRun{
				Id:      "test-run-123",
				Name:    "test-suite",
				Trigger: "local",
			}

			err := reporter.Report(context.Background(), suiteRun, tt.summary)
			if err != nil {
				t.Fatalf("Report() error = %v", err)
			}

			output := buf.String()

			for _, expected := range tt.expectedContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
				}
			}

			// For the no attributes case, verify the attributes section is not present
			if tt.name == "no attributes section when empty" {
				if strings.Contains(output, "--------------ATTRIBUTES---------------") {
					t.Errorf("Expected no ATTRIBUTES section when attributes are empty, but found one.\nFull output:\n%s", output)
				}
			}
		})
	}
}