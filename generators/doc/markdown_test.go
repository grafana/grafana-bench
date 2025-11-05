package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-bench/pkg/utils/test/assert"
)

func TestBuildDocVersionRegex(t *testing.T) {
	t.Parallel()

	regex := buildDocVersionRegex()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Latest Version pattern",
			input:    "Latest Version: v1.2.3",
			expected: true,
		},
		{
			name:     "grafana-bench pattern",
			input:    "grafana-bench:v1.2.3",
			expected: true,
		},
		{
			name:     "benchRevision pattern with quotes",
			input:    "benchRevision: 'v1.2.3'",
			expected: true,
		},
		{
			name:     "bench pattern",
			input:    "bench:v1.2.3",
			expected: true,
		},
		{
			name:     "backtick pattern",
			input:    "`v1.2.3`",
			expected: true,
		},
		{
			name:     "version pattern with quotes",
			input:    "version: 'v1.2.3'",
			expected: true,
		},
		{
			name:     "version pattern with double quotes",
			input:    `version: "v1.2.3"`,
			expected: true,
		},
		{
			name:     "version pattern with spaces",
			input:    "version:   'v1.2.3'",
			expected: true,
		},
		{
			name:     "pre-release version",
			input:    "grafana-bench:v1.2.3-alpha.1",
			expected: true,
		},
		{
			name:     "build metadata version",
			input:    "grafana-bench:v1.2.3+build.123",
			expected: true,
		},
		{
			name:     "no match - invalid pattern",
			input:    "random text v1.2.3",
			expected: false,
		},
		{
			name:     "no match - invalid semver",
			input:    "grafana-bench:v1.2",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := regex.MatchString(tc.input)
			assert.Equal(t, "regex match assertion failed", tc.expected, matches)
		})
	}
}

func TestUpdateSemverInMarkdown(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()

	testCases := []struct {
		name           string
		filename       string
		content        string
		newVersion     string
		expectedOutput string
	}{
		{
			name:       "update multiple patterns",
			filename:   "test.md",
			newVersion: "v2.0.0",
			content: `# Test Doc
Latest Version: v1.0.0
grafana-bench:v1.0.0
benchRevision: 'v1.0.0'
bench:v1.0.0
` + "`v1.0.0`" + `
version: 'v1.0.0'
`,
			expectedOutput: `# Test Doc
Latest Version: v2.0.0
grafana-bench:v2.0.0
benchRevision: 'v2.0.0'
bench:v2.0.0
` + "`v2.0.0`" + `
version: 'v2.0.0'
`,
		},
		{
			name:       "no matches",
			filename:   "no_matches.md",
			newVersion: "v2.0.0",
			content: `# Test Doc
This file has no version references.
`,
			expectedOutput: `# Test Doc
This file has no version references.
`,
		},
		{
			name:       "mixed quotes and spacing",
			filename:   "mixed.md",
			newVersion: "v3.1.4",
			content: `version:   "v1.5.2"
benchRevision: 'v1.5.2'
grafana-bench:v1.5.2
`,
			expectedOutput: `version:   "v3.1.4"
benchRevision: 'v3.1.4'
grafana-bench:v3.1.4
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write test file
			filePath := filepath.Join(testDir, tc.filename)
			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Update versions
			err = updateSemverInMarkdown(testDir, tc.newVersion)
			if err != nil {
				t.Fatalf("updateSemverInMarkdown failed: %v", err)
			}

			// Read and verify output
			output, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}
			assert.Equal(t, "content assertion failed", tc.expectedOutput, string(output))
		})
	}
}

func TestUpdateSemverInMarkdown_NonExistentDir(t *testing.T) {
	t.Parallel()

	err := updateSemverInMarkdown("/non/existent/dir", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestUpdateSemverInMarkdown_SkipsPrefixedFiles(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()

	// Create a file with "bench_" prefix (should be skipped)
	prefixedFile := filepath.Join(testDir, "bench_test.md")
	prefixedContent := "grafana-bench:v1.0.0"
	err := os.WriteFile(prefixedFile, []byte(prefixedContent), 0644)
	if err != nil {
		t.Fatalf("failed to write prefixed test file: %v", err)
	}

	// Create a regular file (should be updated)  
	regularFile := filepath.Join(testDir, "regular.md")
	regularContent := "grafana-bench:v1.0.0"
	err = os.WriteFile(regularFile, []byte(regularContent), 0644)
	if err != nil {
		t.Fatalf("failed to write regular test file: %v", err)
	}

	// Update versions
	err = updateSemverInMarkdown(testDir, "v2.0.0")
	if err != nil {
		t.Fatalf("updateSemverInMarkdown failed: %v", err)
	}

	// Verify prefixed file wasn't updated
	prefixedOutput, err := os.ReadFile(prefixedFile)
	if err != nil {
		t.Fatalf("failed to read prefixed file: %v", err)
	}
	assert.Equal(t, "prefixed file should not be updated", prefixedContent, string(prefixedOutput))

	// Verify regular file was updated
	regularOutput, err := os.ReadFile(regularFile)
	if err != nil {
		t.Fatalf("failed to read regular file: %v", err)
	}
	assert.Equal(t, "regular file should be updated", "grafana-bench:v2.0.0", string(regularOutput))
}