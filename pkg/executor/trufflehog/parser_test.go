package trufflehog

import (
	"bytes"
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

func TestParseFindings(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name       string
		testdata   string
		expectErr  error
		checkExtra func(*testing.T, executor.SuiteRunSummary)
	}{
		{
			name:      "JSON with findings",
			testdata:  "./testdata/with-findings.json",
			expectErr: nil,
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				if summary.Attributes == nil {
					t.Fatal("Attributes should not be nil")
				}
				assert.Equal(t, "total findings", "2", summary.Attributes["total_findings"])
				assert.Equal(t, "verified", "1", summary.Attributes["verified"])
				assert.Equal(t, "unverified", "1", summary.Attributes["unverified"])
				assert.Equal(t, "unique detectors", "1", summary.Attributes["unique_detectors"])

				assert.Equal(t, "suite name", "trufflehog", summary.SuiteName)
				assert.Equal(t, "suite status", executor.SuiteFailed, summary.Status)
				assert.Equal(t, "tests executed", int32(2), summary.TestsExecuted)
				assert.Equal(t, "tests failed", int32(1), summary.TestsFailed)
				assert.Equal(t, "tests passed", int32(1), summary.TestsPassed)

				metricNames := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricNames[m.Name] = m.Value
				}
				assert.Equal(t, "total findings metric", 2.0, metricNames["trufflehog_total_findings"])
				assert.Equal(t, "verified metric", 1.0, metricNames["trufflehog_verified"])
				assert.Equal(t, "unverified metric", 1.0, metricNames["trufflehog_unverified"])
				assert.Equal(t, "unique detectors metric", 1.0, metricNames["trufflehog_unique_detectors"])

				scanCompleted := false
				for _, m := range summary.Metrics {
					if m.Name == "trufflehog_scan_completed" {
						scanCompleted = true
						assert.Equal(t, "scan completed value", 1.0, m.Value)
					}
				}
				if !scanCompleted {
					t.Error("Missing trufflehog_scan_completed metric")
				}

				detectorFindings := 0
				for _, m := range summary.Metrics {
					if m.Name == "trufflehog_detector_findings" {
						detectorFindings++
						assert.Equal(t, "detector label", "Github", m.Labels["detector"])
						if _, ok := m.Labels["verified"]; !ok {
							t.Error("trufflehog_detector_findings missing verified label")
						}
					}
				}
				// expect 2 entries: one for verified=true, one for verified=false
				if detectorFindings != 2 {
					t.Errorf("Expected 2 detector_findings metrics (verified+unverified), got %d", detectorFindings)
				}

				if len(summary.TestRuns) != 2 {
					t.Fatalf("Expected 2 test runs, got %d", len(summary.TestRuns))
				}
				first := summary.TestRuns[0]
				assert.Equal(t, "first test file", "testfile", first.TestFile)
				assert.Equal(t, "first detector", "Github", first.Attributes["detector"])
				assert.Equal(t, "first verified", "true", first.Attributes["verified"])
				assert.Equal(t, "first username", "test-user", first.Attributes["username"])
				assert.Equal(t, "first name", "Test User", first.Attributes["name"])
				assert.Equal(t, "first url", "https://github.com/test-user", first.Attributes["url"])

				// unverified finding has no username/name/url — should not be present
				second := summary.TestRuns[1]
				if _, ok := second.Attributes["username"]; ok {
					t.Error("unverified finding should not have username attribute")
				}
				if _, ok := second.Attributes["name"]; ok {
					t.Error("unverified finding should not have name attribute")
				}
				if _, ok := second.Attributes["url"]; ok {
					t.Error("unverified finding should not have url attribute")
				}
			},
		},
		{
			name:      "JSON with no findings",
			testdata:  "./testdata/no-findings.json",
			expectErr: nil,
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				assert.Equal(t, "suite name", "trufflehog", summary.SuiteName)
				assert.Equal(t, "suite status", executor.SuitePassed, summary.Status)
				assert.Equal(t, "tests executed", int32(0), summary.TestsExecuted)
				assert.Equal(t, "total findings", "0", summary.Attributes["total_findings"])
				assert.Equal(t, "verified", "0", summary.Attributes["verified"])
				assert.Equal(t, "unverified", "0", summary.Attributes["unverified"])

				metricNames := make(map[string]float64)
				for _, m := range summary.Metrics {
					metricNames[m.Name] = m.Value
				}
				assert.Equal(t, "total findings metric", 0.0, metricNames["trufflehog_total_findings"])
				assert.Equal(t, "verified metric", 0.0, metricNames["trufflehog_verified"])
				assert.Equal(t, "unverified metric", 0.0, metricNames["trufflehog_unverified"])

				scanCompleted := false
				for _, m := range summary.Metrics {
					if m.Name == "trufflehog_scan_completed" {
						scanCompleted = true
					}
				}
				if !scanCompleted {
					t.Error("Missing trufflehog_scan_completed metric")
				}

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
				t.Fatalf("Could not read test data file: %v", err)
			}

			summary, err := ParseFindings(bytes.NewBuffer(report), "", nil)
			if tc.expectErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tc.expectErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFindings: %v", err)
			}

			if tc.checkExtra != nil {
				tc.checkExtra(t, summary)
			}
		})
	}
}

func TestParseFindings_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseFindings(bytes.NewBufferString(`{"not": "an array"`), "", nil)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}
