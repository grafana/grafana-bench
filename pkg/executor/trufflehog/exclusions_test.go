package trufflehog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

const orgOnlyExclusions = `# Test patterns
(?i)node_modules/
vendor/
\.git/

# Generated files
.*\.pb\.go$
.*_gen\.go$
`

const orgAndRepoExclusions = `# Test patterns
(?i)node_modules/
vendor/
\.git/

# Generated files
.*\.pb\.go$
.*_gen\.go$
# Repo-local .trufflehogignore
config/secrets\.yaml
tests/fixtures/.*\.key$
`

func TestParseExclusions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		input              string
		wantTotal          int
		wantOrg            int
		wantRepo           int
		wantRepoPatterns   []string
		wantOrgCategories  map[string]int
		wantRepoCategories map[string]int
	}{
		{
			name:      "org-wide only",
			input:     orgOnlyExclusions,
			wantTotal: 5,
			wantOrg:   5,
			wantRepo:  0,
			wantOrgCategories: map[string]int{
				"Test patterns":   3,
				"Generated files": 2,
			},
			wantRepoCategories: map[string]int{},
		},
		{
			name:             "org + repo-local",
			input:            orgAndRepoExclusions,
			wantTotal:        7,
			wantOrg:          5,
			wantRepo:         2,
			wantRepoPatterns: []string{"config/secrets\\.yaml", "tests/fixtures/.*\\.key$"},
			wantOrgCategories: map[string]int{
				"Test patterns":   3,
				"Generated files": 2,
			},
			wantRepoCategories: map[string]int{
				"repo-local": 2,
			},
		},
		{
			name:               "empty input",
			input:              "",
			wantTotal:          0,
			wantOrg:            0,
			wantRepo:           0,
			wantOrgCategories:  map[string]int{},
			wantRepoCategories: map[string]int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stats, err := ParseExclusions(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseExclusions: %v", err)
			}

			assert.Equal(t, "total patterns", tc.wantTotal, stats.TotalPatterns)
			assert.Equal(t, "org patterns", tc.wantOrg, stats.OrgPatterns)
			assert.Equal(t, "repo patterns", tc.wantRepo, stats.RepoPatterns)

			if tc.wantRepoPatterns != nil {
				assert.Equal(t, "repo pattern count", len(tc.wantRepoPatterns), len(stats.RepoPatternList))
				for i, want := range tc.wantRepoPatterns {
					if i < len(stats.RepoPatternList) {
						assert.Equal(t, "repo pattern", want, stats.RepoPatternList[i])
					}
				}
			}

			for cat, count := range tc.wantOrgCategories {
				assert.Equal(t, "org category "+cat, count, stats.OrgCategories[cat])
			}
			for cat, count := range tc.wantRepoCategories {
				assert.Equal(t, "repo category "+cat, count, stats.RepoCategories[cat])
			}
		})
	}
}

func TestParseExclusionFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "exclude.txt")
	err := os.WriteFile(path, []byte(orgAndRepoExclusions), 0o644)
	if err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	stats, err := ParseExclusionFile(path)
	if err != nil {
		t.Fatalf("ParseExclusionFile: %v", err)
	}

	assert.Equal(t, "total patterns", 7, stats.TotalPatterns)
	assert.Equal(t, "org patterns", 5, stats.OrgPatterns)
	assert.Equal(t, "repo patterns", 2, stats.RepoPatterns)
}

func TestParseExclusionFile_NotFound(t *testing.T) {
	t.Parallel()

	_, err := ParseExclusionFile("/nonexistent/path/exclude.txt")
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}

func TestExclusionStats_Metrics(t *testing.T) {
	t.Parallel()

	stats := &ExclusionStats{
		TotalPatterns: 7,
		OrgPatterns:   5,
		RepoPatterns:  2,
		OrgCategories: map[string]int{
			"Test patterns":   3,
			"Generated files": 2,
		},
		RepoCategories: map[string]int{
			"repo-local": 2,
		},
		RepoPatternList: []string{
			"config/secrets\\.yaml",
			"tests/fixtures/.*\\.key$",
		},
	}

	m := stats.Metrics(1000)

	byName := make(map[string][]float64)
	for _, metric := range m {
		byName[metric.Name] = append(byName[metric.Name], metric.Value)
	}

	assert.Equal(t, "total metric count", 1, len(byName["trufflehog_exclusion_patterns_total"]))
	assert.Equal(t, "total value", 7.0, byName["trufflehog_exclusion_patterns_total"][0])

	assert.Equal(t, "org metric count", 1, len(byName["trufflehog_exclusion_patterns_org"]))
	assert.Equal(t, "org value", 5.0, byName["trufflehog_exclusion_patterns_org"][0])

	assert.Equal(t, "repo metric count", 1, len(byName["trufflehog_exclusion_patterns_repo"]))
	assert.Equal(t, "repo value", 2.0, byName["trufflehog_exclusion_patterns_repo"][0])

	// 2 org categories + 1 repo category = 3 by_category metrics
	assert.Equal(t, "category metric count", 3, len(byName["trufflehog_exclusion_patterns_by_category"]))

	// 2 repo exclusion pattern metrics
	assert.Equal(t, "repo exclusion count", 2, len(byName["trufflehog_repo_exclusion"]))

	repoExclPatterns := make(map[string]bool)
	for _, metric := range m {
		if metric.Name == "trufflehog_repo_exclusion" {
			repoExclPatterns[metric.Labels["pattern"]] = true
			assert.Equal(t, "repo exclusion value", 1.0, metric.Value)
		}
	}
	if !repoExclPatterns["config/secrets\\.yaml"] {
		t.Error("missing repo exclusion pattern: config/secrets\\.yaml")
	}
	if !repoExclPatterns["tests/fixtures/.*\\.key$"] {
		t.Error("missing repo exclusion pattern: tests/fixtures/.*\\.key$")
	}
}
