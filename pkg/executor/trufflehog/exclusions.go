package trufflehog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grafana/grafana-bench/pkg/metrics"
)

const repoLocalMarker = "# Repo-local .trufflehogignore"

// ExclusionStats holds counts derived from a TruffleHog exclude-paths file.
type ExclusionStats struct {
	TotalPatterns   int
	OrgPatterns     int
	RepoPatterns    int
	OrgCategories   map[string]int
	RepoCategories  map[string]int
	RepoPatternList []string
}

// ParseExclusionFile opens path and delegates to ParseExclusions.
func ParseExclusionFile(path string) (*ExclusionStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening exclusion file: %w", err)
	}
	defer f.Close()
	return ParseExclusions(f)
}

// ParseExclusions reads a TruffleHog exclude-paths file and returns stats.
//
// The file uses Go regexp syntax, one pattern per line. Comments (#) and blank
// lines are ignored when counting patterns. If the file contains the marker
// "# Repo-local .trufflehogignore", patterns after that line are counted as
// repo-local; patterns before it are org-wide.
//
// Category comments (lines starting with "# " that are not the repo-local
// marker) are tracked: every subsequent pattern inherits the most recent
// category comment until a new one appears.
func ParseExclusions(r io.Reader) (*ExclusionStats, error) {
	scanner := bufio.NewScanner(r)

	stats := &ExclusionStats{
		OrgCategories:  make(map[string]int),
		RepoCategories: make(map[string]int),
	}

	inRepoLocal := false
	currentCategory := "uncategorized"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if line == repoLocalMarker {
			inRepoLocal = true
			currentCategory = "repo-local"
			continue
		}

		if strings.HasPrefix(line, "#") {
			currentCategory = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		stats.TotalPatterns++
		if inRepoLocal {
			stats.RepoPatterns++
			stats.RepoCategories[currentCategory]++
			stats.RepoPatternList = append(stats.RepoPatternList, line)
		} else {
			stats.OrgPatterns++
			stats.OrgCategories[currentCategory]++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading exclusion file: %w", err)
	}

	return stats, nil
}

// Metrics converts ExclusionStats into a slice of metrics suitable for
// appending to a SuiteRunSummary.
func (s *ExclusionStats) Metrics(timestamp int64) []metrics.Metric {
	m := []metrics.Metric{
		{
			Name:      "trufflehog_exclusion_patterns_total",
			Value:     float64(s.TotalPatterns),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
		{
			Name:      "trufflehog_exclusion_patterns_org",
			Value:     float64(s.OrgPatterns),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
		{
			Name:      "trufflehog_exclusion_patterns_repo",
			Value:     float64(s.RepoPatterns),
			Labels:    map[string]string{},
			Timestamp: timestamp,
		},
	}

	for category, count := range s.OrgCategories {
		m = append(m, metrics.Metric{
			Name:  "trufflehog_exclusion_patterns_by_category",
			Value: float64(count),
			Labels: map[string]string{
				"category": category,
				"source":   "org",
			},
			Timestamp: timestamp,
		})
	}

	for category, count := range s.RepoCategories {
		m = append(m, metrics.Metric{
			Name:  "trufflehog_exclusion_patterns_by_category",
			Value: float64(count),
			Labels: map[string]string{
				"category": category,
				"source":   "repo",
			},
			Timestamp: timestamp,
		})
	}

	for _, pattern := range s.RepoPatternList {
		m = append(m, metrics.Metric{
			Name:  "trufflehog_repo_exclusion",
			Value: 1.0,
			Labels: map[string]string{
				"pattern": pattern,
			},
			Timestamp: timestamp,
		})
	}

	return m
}
