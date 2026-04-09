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
		name        string
		testdata    string
		excludeFile string
		expectErr   error
		checkExtra  func(*testing.T, executor.SuiteRunSummary)
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

				assertNoExclusionMetrics(t, summary)
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

				assertNoExclusionMetrics(t, summary)
			},
		},
		{
			name:        "findings with org and repo exclusions",
			testdata:    "./testdata/with-findings.json",
			excludeFile: "./testdata/exclude-org-and-repo.txt",
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				// Scan metrics should still be correct
				assert.Equal(t, "suite status", executor.SuiteFailed, summary.Status)
				assert.Equal(t, "tests executed", int32(2), summary.TestsExecuted)
				assert.Equal(t, "total findings", "2", summary.Attributes["total_findings"])

				metricsByName := collectMetrics(t, summary)

				assert.Equal(t, "total findings metric", 2.0, metricsByName["trufflehog_total_findings"][0])
				assert.Equal(t, "scan completed", 1.0, metricsByName["trufflehog_scan_completed"][0])

				// Exclusion aggregate metrics
				assert.Equal(t, "exclusion total", 7.0, metricsByName["trufflehog_exclusion_patterns_total"][0])
				assert.Equal(t, "exclusion org", 5.0, metricsByName["trufflehog_exclusion_patterns_org"][0])
				assert.Equal(t, "exclusion repo", 2.0, metricsByName["trufflehog_exclusion_patterns_repo"][0])

				// Category breakdown: 2 org categories + 1 repo category
				assert.Equal(t, "by_category count", 3, len(metricsByName["trufflehog_exclusion_patterns_by_category"]))
				assertCategoryMetrics(t, summary, map[string]map[string]float64{
					"org":  {"Dependencies": 3.0, "Generated files": 2.0},
					"repo": {"repo-local": 2.0},
				})

				// Individual repo exclusion patterns
				assert.Equal(t, "repo_exclusion count", 2, len(metricsByName["trufflehog_repo_exclusion"]))
				assertRepoExclusionPatterns(t, summary, []string{
					"integrations/.*/vendor/",
					"config/secrets\\.yaml",
				})
			},
		},
		{
			name:        "no findings with org and repo exclusions",
			testdata:    "./testdata/no-findings.json",
			excludeFile: "./testdata/exclude-org-and-repo.txt",
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				assert.Equal(t, "suite status", executor.SuitePassed, summary.Status)
				assert.Equal(t, "tests executed", int32(0), summary.TestsExecuted)

				metricsByName := collectMetrics(t, summary)

				assert.Equal(t, "total findings metric", 0.0, metricsByName["trufflehog_total_findings"][0])
				assert.Equal(t, "scan completed", 1.0, metricsByName["trufflehog_scan_completed"][0])

				// Exclusion metrics are present regardless of findings
				assert.Equal(t, "exclusion total", 7.0, metricsByName["trufflehog_exclusion_patterns_total"][0])
				assert.Equal(t, "exclusion org", 5.0, metricsByName["trufflehog_exclusion_patterns_org"][0])
				assert.Equal(t, "exclusion repo", 2.0, metricsByName["trufflehog_exclusion_patterns_repo"][0])

				assertRepoExclusionPatterns(t, summary, []string{
					"integrations/.*/vendor/",
					"config/secrets\\.yaml",
				})
			},
		},
		{
			name:        "no findings with org-only exclusions",
			testdata:    "./testdata/no-findings.json",
			excludeFile: "./testdata/exclude-org-only.txt",
			checkExtra: func(t *testing.T, summary executor.SuiteRunSummary) {
				metricsByName := collectMetrics(t, summary)

				assert.Equal(t, "exclusion total", 5.0, metricsByName["trufflehog_exclusion_patterns_total"][0])
				assert.Equal(t, "exclusion org", 5.0, metricsByName["trufflehog_exclusion_patterns_org"][0])
				assert.Equal(t, "exclusion repo", 0.0, metricsByName["trufflehog_exclusion_patterns_repo"][0])

				// No repo exclusion metrics when there are no repo-local patterns
				if len(metricsByName["trufflehog_repo_exclusion"]) != 0 {
					t.Errorf("expected no repo_exclusion metrics for org-only file, got %d",
						len(metricsByName["trufflehog_repo_exclusion"]))
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

			summary, err := ParseFindings(bytes.NewBuffer(report), tc.excludeFile, nil)
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

// collectMetrics groups all metrics in the summary by name.
func collectMetrics(t *testing.T, summary executor.SuiteRunSummary) map[string][]float64 {
	t.Helper()
	m := make(map[string][]float64)
	for _, metric := range summary.Metrics {
		m[metric.Name] = append(m[metric.Name], metric.Value)
	}
	return m
}

// assertNoExclusionMetrics verifies that no exclusion-related metrics exist.
func assertNoExclusionMetrics(t *testing.T, summary executor.SuiteRunSummary) {
	t.Helper()
	for _, m := range summary.Metrics {
		switch m.Name {
		case "trufflehog_exclusion_patterns_total",
			"trufflehog_exclusion_patterns_org",
			"trufflehog_exclusion_patterns_repo",
			"trufflehog_exclusion_patterns_by_category",
			"trufflehog_repo_exclusion":
			t.Errorf("unexpected exclusion metric %q when no exclude file provided", m.Name)
		}
	}
}

// assertCategoryMetrics verifies the by_category metrics match expected source/category/value.
func assertCategoryMetrics(t *testing.T, summary executor.SuiteRunSummary, expected map[string]map[string]float64) {
	t.Helper()
	actual := make(map[string]map[string]float64)
	for _, m := range summary.Metrics {
		if m.Name == "trufflehog_exclusion_patterns_by_category" {
			source := m.Labels["source"]
			if actual[source] == nil {
				actual[source] = make(map[string]float64)
			}
			actual[source][m.Labels["category"]] = m.Value
		}
	}
	for source, categories := range expected {
		for category, value := range categories {
			got, ok := actual[source][category]
			if !ok {
				t.Errorf("missing by_category metric: source=%s category=%s", source, category)
				continue
			}
			assert.Equal(t, source+" "+category, value, got)
		}
	}
}

// assertRepoExclusionPatterns verifies each expected pattern has a repo_exclusion metric.
func assertRepoExclusionPatterns(t *testing.T, summary executor.SuiteRunSummary, expected []string) {
	t.Helper()
	found := make(map[string]bool)
	for _, m := range summary.Metrics {
		if m.Name == "trufflehog_repo_exclusion" {
			found[m.Labels["pattern"]] = true
			assert.Equal(t, "repo_exclusion value for "+m.Labels["pattern"], 1.0, m.Value)
		}
	}
	for _, pattern := range expected {
		if !found[pattern] {
			t.Errorf("missing repo_exclusion metric for pattern: %s", pattern)
		}
	}
}
