package trufflehog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

func TestParseExclusions(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name  string
		input string
		check func(*testing.T, *ExclusionStats)
	}{
		{
			name: "org-only patterns",
			input: `# Go vendor directory (third-party code, not repo secrets)
vendor/

# Lock files and checksums (contain hashes, not secrets)
go\.sum$
go\.mod$

# Dependency manifests (contain URLs / hashes that trigger false positives)
package\.json$
package-lock\.json$
yarn\.lock$
`,
			check: func(t *testing.T, stats *ExclusionStats) {
				assert.Equal(t, "total patterns", 6, stats.TotalPatterns)
				assert.Equal(t, "org patterns", 6, stats.OrgPatterns)
				assert.Equal(t, "repo patterns", 0, stats.RepoPatterns)
				assert.Equal(t, "org categories count", 3, len(stats.OrgCategories))
				assert.Equal(t, "repo categories count", 0, len(stats.RepoCategories))
				assert.Equal(t, "repo pattern list length", 0, len(stats.RepoPatternList))
			},
		},
		{
			name: "org + repo-local patterns",
			input: `# Go vendor directory
vendor/

# Lock files
go\.sum$
go\.mod$

# Repo-local .trufflehogignore
testdata/
fixtures/fake-keys\.pem$
`,
			check: func(t *testing.T, stats *ExclusionStats) {
				assert.Equal(t, "total patterns", 5, stats.TotalPatterns)
				assert.Equal(t, "org patterns", 3, stats.OrgPatterns)
				assert.Equal(t, "repo patterns", 2, stats.RepoPatterns)
				assert.Equal(t, "repo pattern list length", 2, len(stats.RepoPatternList))
				assert.Equal(t, "first repo pattern", "testdata/", stats.RepoPatternList[0])
				assert.Equal(t, "second repo pattern", `fixtures/fake-keys\.pem$`, stats.RepoPatternList[1])
			},
		},
		{
			name:  "empty file",
			input: "# Empty exclusion file — no patterns.\n",
			check: func(t *testing.T, stats *ExclusionStats) {
				assert.Equal(t, "total patterns", 0, stats.TotalPatterns)
				assert.Equal(t, "org patterns", 0, stats.OrgPatterns)
				assert.Equal(t, "repo patterns", 0, stats.RepoPatterns)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stats, err := ParseExclusions(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseExclusions: %v", err)
			}

			tc.check(t, stats)
		})
	}
}

func TestParseExclusionFile(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "exclude.txt")
	err := os.WriteFile(tmp, []byte("# vendor\nvendor/\ngo\\.sum$\n"), 0644)
	if err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	stats, err := ParseExclusionFile(tmp)
	if err != nil {
		t.Fatalf("ParseExclusionFile: %v", err)
	}
	assert.Equal(t, "total patterns", 2, stats.TotalPatterns)
}

func TestParseExclusionFile_NotFound(t *testing.T) {
	t.Parallel()

	_, err := ParseExclusionFile(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}

func TestExclusionStats_Metrics(t *testing.T) {
	t.Parallel()

	stats := &ExclusionStats{
		TotalPatterns: 5,
		OrgPatterns:   3,
		RepoPatterns:  2,
		OrgCategories: map[string]int{
			"vendor": 1,
			"locks":  2,
		},
		RepoCategories: map[string]int{
			"repo-local": 2,
		},
		RepoPatternList: []string{"migration_data/", `secrets\.env$`},
	}

	m := stats.Metrics(1234567890)

	metricMap := make(map[string]float64)
	for _, metric := range m {
		metricMap[metric.Name] = metric.Value
	}

	assert.Equal(t, "total metric", 5.0, metricMap["trufflehog_exclusion_patterns_total"])
	assert.Equal(t, "org metric", 3.0, metricMap["trufflehog_exclusion_patterns_org"])
	assert.Equal(t, "repo metric", 2.0, metricMap["trufflehog_exclusion_patterns_repo"])

	categoryMetrics := 0
	for _, metric := range m {
		if metric.Name == "trufflehog_exclusion_patterns_by_category" {
			categoryMetrics++
			assert.Equal(t, "timestamp", int64(1234567890), metric.Timestamp)
		}
	}
	assert.Equal(t, "category metrics count", 3, categoryMetrics)

	patternMetrics := []string{}
	for _, metric := range m {
		if metric.Name == "trufflehog_repo_exclusion" {
			patternMetrics = append(patternMetrics, metric.Labels["pattern"])
			assert.Equal(t, "pattern metric value", 1.0, metric.Value)
		}
	}
	assert.Equal(t, "pattern metrics count", 2, len(patternMetrics))
}
