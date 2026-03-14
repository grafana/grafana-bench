package gotest

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

func TestParseJsonOutput(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		testdata  string
		expect    executor.SuiteRunSummary
		expectErr error
	}{
		{
			name:      "file with valid format",
			testdata:  "./testdata/output.json",
			expectErr: nil,
			expect: executor.SuiteRunSummary{
				Status:            "",
				TestsExecuted:     7,
				TestsFailed:       1,
				TestsFlaky:        0,
				TestsPassed:       6,
				TestsError:        0,
				TotalDuration:     time.Duration(204968132),
				ScenariosDuration: time.Duration(0.22 * float32(time.Second)),
				TestRuns: []executor.TestRunSummary{
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestFailing",
						Status:           executor.TestFailed,
						ScenarioDuration: time.Duration(float64(0.1) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.1) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestPassing1",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.02) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.02) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestPassing2",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.06) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.06) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestPassing3",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.02) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.02) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestPassing3/SubTest1",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.01) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.01) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests",
						TestFile:         "TestPassing3/SubTest2",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.01) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.01) * float64(time.Second)),
					},
					{
						TestFolder:       "github.com/grafana/grafana-bench/pkg/executor/gotest/tests/subpkg",
						TestFile:         "TestPassing4",
						Status:           executor.TestPassed,
						ScenarioDuration: time.Duration(float64(0.02) * float64(time.Second)),
						TotalDuration:    time.Duration(float64(0.02) * float64(time.Second)),
					},
				},
			},
		},
		{
			name:      "file with invalid format returns an error",
			testdata:  "./testdata/output_invalid.json",
			expectErr: ErrInvalidFormat,
		},
		{
			name:     "empty test suite returns zero duration",
			testdata: "./testdata/output_no_tests.json",
			expect: executor.SuiteRunSummary{
				TestsExecuted: 0,
				TotalDuration: 0,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report, err := os.ReadFile(tc.testdata)
			if err != nil {
				t.Errorf("Could not read test data file: %v", err.Error())
			}

			summary, err := ParseJsonOutput(bytes.NewBuffer(report))
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("Expected %v got %v", tc.expectErr, err)
			}

			assert.SuiteSummaryEqual(t, &tc.expect, summary)
		})
	}
}
