package trufflehog

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestParseFindings_WithExcludeFile(t *testing.T) {
	t.Parallel()

	excludeContent := `# Dependencies
(?i)node_modules/
vendor/
\.git/

# Generated files
.*\.pb\.go$
.*_gen\.go$
# Repo-local .trufflehogignore
integrations/.*/vendor/
config/secrets\.yaml
`
	dir := t.TempDir()
	excludePath := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(excludePath, []byte(excludeContent), 0o644); err != nil {
		t.Fatalf("writing exclude file: %v", err)
	}

	summary, err := ParseFindings(bytes.NewBufferString("[]"), excludePath, nil)
	if err != nil {
		t.Fatalf("ParseFindings: %v", err)
	}

	metricsByName := make(map[string][]float64)
	for _, m := range summary.Metrics {
		metricsByName[m.Name] = append(metricsByName[m.Name], m.Value)
	}

	assert.Equal(t, "exclusion total count", 1, len(metricsByName["trufflehog_exclusion_patterns_total"]))
	assert.Equal(t, "exclusion total value", 7.0, metricsByName["trufflehog_exclusion_patterns_total"][0])

	assert.Equal(t, "exclusion org count", 1, len(metricsByName["trufflehog_exclusion_patterns_org"]))
	assert.Equal(t, "exclusion org value", 5.0, metricsByName["trufflehog_exclusion_patterns_org"][0])

	assert.Equal(t, "exclusion repo count", 1, len(metricsByName["trufflehog_exclusion_patterns_repo"]))
	assert.Equal(t, "exclusion repo value", 2.0, metricsByName["trufflehog_exclusion_patterns_repo"][0])

	// 2 org categories (Dependencies, Generated files) + 1 repo category (repo-local)
	assert.Equal(t, "by_category metric count", 3, len(metricsByName["trufflehog_exclusion_patterns_by_category"]))

	// Verify category labels
	categorySources := make(map[string]map[string]float64)
	for _, m := range summary.Metrics {
		if m.Name == "trufflehog_exclusion_patterns_by_category" {
			source := m.Labels["source"]
			if categorySources[source] == nil {
				categorySources[source] = make(map[string]float64)
			}
			categorySources[source][m.Labels["category"]] = m.Value
		}
	}
	assert.Equal(t, "org Dependencies count", 3.0, categorySources["org"]["Dependencies"])
	assert.Equal(t, "org Generated files count", 2.0, categorySources["org"]["Generated files"])
	assert.Equal(t, "repo repo-local count", 2.0, categorySources["repo"]["repo-local"])

	// Verify individual repo exclusion pattern metrics
	assert.Equal(t, "repo_exclusion metric count", 2, len(metricsByName["trufflehog_repo_exclusion"]))

	repoPatterns := make(map[string]bool)
	for _, m := range summary.Metrics {
		if m.Name == "trufflehog_repo_exclusion" {
			repoPatterns[m.Labels["pattern"]] = true
			assert.Equal(t, "repo_exclusion value", 1.0, m.Value)
		}
	}
	if !repoPatterns["integrations/.*/vendor/"] {
		t.Error("missing repo exclusion pattern: integrations/.*/vendor/")
	}
	if !repoPatterns["config/secrets\\.yaml"] {
		t.Error("missing repo exclusion pattern: config/secrets\\.yaml")
	}

	// Scan metrics should still be present alongside exclusion metrics
	assert.Equal(t, "scan completed present", 1, len(metricsByName["trufflehog_scan_completed"]))
}

func TestParseFindings_WithOrgOnlyExcludeFile(t *testing.T) {
	t.Parallel()

	excludeContent := `# Dependencies
(?i)node_modules/
vendor/
`
	dir := t.TempDir()
	excludePath := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(excludePath, []byte(excludeContent), 0o644); err != nil {
		t.Fatalf("writing exclude file: %v", err)
	}

	summary, err := ParseFindings(bytes.NewBufferString("[]"), excludePath, nil)
	if err != nil {
		t.Fatalf("ParseFindings: %v", err)
	}

	metricsByName := make(map[string]float64)
	for _, m := range summary.Metrics {
		if len(metricsByName) == 0 || m.Labels == nil || len(m.Labels) == 0 {
			metricsByName[m.Name] = m.Value
		}
	}

	assert.Equal(t, "exclusion total", 2.0, metricsByName["trufflehog_exclusion_patterns_total"])
	assert.Equal(t, "exclusion org", 2.0, metricsByName["trufflehog_exclusion_patterns_org"])
	assert.Equal(t, "exclusion repo", 0.0, metricsByName["trufflehog_exclusion_patterns_repo"])

	// No repo exclusion pattern metrics should exist
	for _, m := range summary.Metrics {
		if m.Name == "trufflehog_repo_exclusion" {
			t.Errorf("unexpected trufflehog_repo_exclusion metric for org-only file: %v", m)
		}
	}
}

func TestParseFindings_WithoutExcludeFile(t *testing.T) {
	t.Parallel()

	summary, err := ParseFindings(bytes.NewBufferString("[]"), "", nil)
	if err != nil {
		t.Fatalf("ParseFindings: %v", err)
	}

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

func TestParseFindings_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseFindings(bytes.NewBufferString(`{"not": "an array"`), "", nil)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}
