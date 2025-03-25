package gotest

import (
	"bytes"
	"os"
	"testing"
)

func TestParseJsonOutput(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                 string
		testdata             string
		expectSummarySuccess bool
		expectSummaryFailure bool
		expectErr            bool
	}{
		{
			name:                 "file with valid format and all passing tests returns the summary",
			testdata:             "./testdata/output.json",
			expectSummarySuccess: true,
			expectSummaryFailure: false,
			expectErr:            false,
		},
		{
			name:                 "file with valid format and failing tests returns the summary",
			testdata:             "./testdata/output_fail.json",
			expectSummarySuccess: true,
			expectSummaryFailure: true,
			expectErr:            false,
		},
		{
			name:                 "file with invalid format returns an error",
			testdata:             "./testdata/output_invalid.json",
			expectSummarySuccess: false,
			expectSummaryFailure: false,
			expectErr:            true,
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
			if tc.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err.Error())
			}

			if tc.expectSummarySuccess && summary.TestsPassed == 0 {
				t.Errorf("Expected test successes in summary but got 0")
			}
			if tc.expectSummaryFailure && summary.TestsFailed == 0 {
				t.Errorf("Expected test failures in summary but got 0")
			}
		})
	}
}
