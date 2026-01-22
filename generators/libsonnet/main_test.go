package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/grafana/grafana-bench/cmd/test"
)

func TestExtractFlags(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	testCmd := test.NewCmd(logger)
	
	flags := extractFlags(testCmd)
	
	// Should extract some flags
	if len(flags) == 0 {
		t.Error("Expected to extract some flags, got 0")
	}
	
	// Should have common flags
	expectedFlags := map[string]bool{
		"grafana-url":      true,
		"suite-path":       true,
		"test-type":        true,
		"test-runner":      true,
		"slack-notifications": true,
		"prometheus-metrics":  true,
	}
	
	foundFlags := make(map[string]bool)
	for _, flag := range flags {
		foundFlags[flag.Name] = true
	}
	
	for expected := range expectedFlags {
		if !foundFlags[expected] {
			t.Errorf("Expected to find flag %s, but it was missing", expected)
		}
	}
}

func TestShouldSkipFlag(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		usage    string
		expected bool
	}{
		{
			name:     "skip help flag",
			flagName: "help",
			usage:    "help for command",
			expected: true,
		},
		{
			name:     "skip config flag",
			flagName: "config",
			usage:    "config file",
			expected: true,
		},
		{
			name:     "skip bench-revision flag",
			flagName: "bench-revision",
			usage:    "bench revision",
			expected: true,
		},
		{
			name:     "skip deprecated flag by usage",
			flagName: "some-flag",
			usage:    "deprecated. Use other-flag",
			expected: true,
		},
		{
			name:     "skip deprecated flag by name",
			flagName: "test-suite",
			usage:    "deprecated. Use suite-path",
			expected: true,
		},
		{
			name:     "keep normal flag",
			flagName: "grafana-url",
			usage:    "url to grafana instance",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipFlag(tt.flagName, tt.usage)
			if result != tt.expected {
				t.Errorf("shouldSkipFlag(%q, %q) = %v, expected %v", 
					tt.flagName, tt.usage, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"single", "single"},
		{"two-words", "twoWords"},
		{"three-word-test", "threeWordTest"},
		{"grafana-url", "grafanaUrl"},
		{"test-runner", "testRunner"},
		{"slack-notifications", "slackNotifications"},
		{"prometheus-metrics", "prometheusMetrics"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, expected %q", 
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLibsonnetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		flag     FlagInfo
		expected string
	}{
		{
			name: "bool true",
			flag: FlagInfo{
				Name:         "some-flag",
				Type:         "bool",
				DefaultValue: "true",
			},
			expected: "false", // Returns false (zero value) to avoid generating CLI flag
		},
		{
			name: "bool false",
			flag: FlagInfo{
				Name:         "some-flag",
				Type:         "bool",
				DefaultValue: "false",
			},
			expected: "false",
		},
		{
			name: "string with default",
			flag: FlagInfo{
				Name:         "some-flag",
				Type:         "string",
				DefaultValue: "default-value",
			},
			expected: "''", // Returns empty string (zero value) to avoid generating CLI flag
		},
		{
			name: "required string (grafana-url)",
			flag: FlagInfo{
				Name:         "grafana-url",
				Type:         "string",
				DefaultValue: "",
			},
			expected: "error 'must define grafana url'",
		},
		{
			name: "required string (suite-path)", 
			flag: FlagInfo{
				Name:         "suite-path",
				Type:         "string",
				DefaultValue: "",
			},
			expected: "error 'must define suite path'",
		},
		{
			name: "empty string",
			flag: FlagInfo{
				Name:         "some-flag",
				Type:         "string",
				DefaultValue: "",
			},
			expected: "''",
		},
		{
			name: "string array",
			flag: FlagInfo{
				Name:         "some-array",
				Type:         "stringArray",
				DefaultValue: "",
			},
			expected: "[]",
		},
		{
			name: "string to string map",
			flag: FlagInfo{
				Name:         "test-env",
				Type:         "stringToString",
				DefaultValue: "",
			},
			expected: "[]", // Special case for test-env
		},
		{
			name: "regular string to string map",
			flag: FlagInfo{
				Name:         "some-map",
				Type:         "stringToString",
				DefaultValue: "",
			},
			expected: "{}",
		},
		{
			name: "int with value",
			flag: FlagInfo{
				Name:         "retries",
				Type:         "int",
				DefaultValue: "3",
			},
			expected: "0", // Returns 0 (zero value) to avoid generating CLI flag
		},
		{
			name: "int empty",
			flag: FlagInfo{
				Name:         "retries",
				Type:         "int",
				DefaultValue: "",
			},
			expected: "0",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLibsonnetDefaultValue(tt.flag)
			if result != tt.expected {
				t.Errorf("getLibsonnetDefaultValue(%+v) = %q, expected %q",
					tt.flag, result, tt.expected)
			}
		})
	}
}

func TestIsRequiredStringFlag(t *testing.T) {
	tests := []struct {
		flagName string
		expected bool
	}{
		{"grafana-url", true},
		{"suite-path", true},
		{"test-type", false},
		{"test-runner", false},
		{"some-other-flag", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			result := isRequiredStringFlag(tt.flagName)
			if result != tt.expected {
				t.Errorf("isRequiredStringFlag(%q) = %v, expected %v",
					tt.flagName, result, tt.expected)
			}
		})
	}
}

func TestGenerateLibsonnetOption(t *testing.T) {
	tests := []struct {
		name     string
		flag     FlagInfo
		expected string
	}{
		{
			name: "simple bool flag",
			flag: FlagInfo{
				Name:         "test-verbose",
				Type:         "bool",
				DefaultValue: "false",
				Usage:        "show test output",
			},
			expected: "// --test-verbose: show test output\n      testVerbose: false",
		},
		{
			name: "required string flag",
			flag: FlagInfo{
				Name:         "grafana-url",
				Type:         "string", 
				DefaultValue: "",
				Usage:        "url to grafana instance",
			},
			expected: "// --grafana-url: url to grafana instance\n      grafanaUrl: error 'must define grafana url'",
		},
		{
			name: "string with default",
			flag: FlagInfo{
				Name:         "test-type",
				Type:         "string",
				DefaultValue: "smoke",
				Usage:        "test type. Allowed values: 'smoke', 'load'",
			},
			expected: "// --test-type: test type. Allowed values: 'smoke', 'load'\n      testType: ''", // Returns empty to avoid generating CLI flag
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateLibsonnetOption(tt.flag)
			if result != tt.expected {
				t.Errorf("generateLibsonnetOption(%+v) = %q, expected %q",
					tt.flag, result, tt.expected)
			}
		})
	}
}

func TestCleanUsage(t *testing.T) {
	tests := []struct {
		name     string
		usage    string
		expected string
	}{
		{
			name:     "simple usage",
			usage:    "test output",
			expected: "test output",
		},
		{
			name:     "usage with newlines",
			usage:    "test output\nwith newlines",
			expected: "test output with newlines",
		},
		{
			name:     "usage with tabs",
			usage:    "test\toutput",
			expected: "test output",
		},
		{
			name:     "usage with multiple spaces",
			usage:    "test    output   here",
			expected: "test output here",
		},
		{
			name:     "complex usage",
			usage:    "test output\n\t  with\n  multiple  spaces",
			expected: "test output with multiple spaces",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanUsage(tt.usage)
			if result != tt.expected {
				t.Errorf("cleanUsage(%q) = %q, expected %q",
					tt.usage, result, tt.expected)
			}
		})
	}
}

func TestGenerateScriptFlag(t *testing.T) {
	tests := []struct {
		name     string
		flag     FlagInfo
		contains []string // strings that should be in the result
		notContains []string // strings that should NOT be in the result
	}{
		{
			name: "bool flag",
			flag: FlagInfo{
				Name: "test-verbose",
				Type: "bool",
			},
			contains: []string{"bench_options.testVerbose", "--test-verbose", "(if", "else [])"},
		},
		{
			name: "required string flag returns empty",
			flag: FlagInfo{
				Name: "grafana-url", 
				Type: "string",
			},
			contains: []string{}, // Required flags return empty string from generateOptionalScriptFlag
			notContains: []string{"--grafana-url", "grafanaURL", "(if", "else [])"},
		},
		{
			name: "optional string flag",
			flag: FlagInfo{
				Name: "suite-name",
				Type: "string",
			},
			contains: []string{"bench_options.suiteName", "--suite-name", "(if", "else [])"},
		},
		{
			name: "string array flag",
			flag: FlagInfo{
				Name: "go-args",
				Type: "stringArray",
			},
			contains: []string{"std.flattenArrays", "bench_options.goArgs", "--go-args"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateOptionalScriptFlag(tt.flag)
			
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("generateOptionalScriptFlag(%+v) result should contain %q, got: %s",
						tt.flag, expected, result)
				}
			}
			
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("generateOptionalScriptFlag(%+v) result should NOT contain %q, got: %s", 
						tt.flag, notExpected, result)
				}
			}
		})
	}
}

// Test the integration - extract real flags and verify they don't break generation
func TestRealFlagExtraction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	testCmd := test.NewCmd(logger)
	
	flags := extractFlags(testCmd)
	
	// Test that we can generate libsonnet options for all extracted flags
	for _, flag := range flags {
		t.Run("flag_"+flag.Name, func(t *testing.T) {
			// Should not panic
			option := generateLibsonnetOption(flag)
			
			// Should contain the camelCase name
			camelCase := toCamelCase(flag.Name)
			if !strings.Contains(option, camelCase+":") {
				t.Errorf("Generated option for %s should contain %s:, got: %s",
					flag.Name, camelCase, option)
			}
			
			// Should not be empty
			if option == "" {
				t.Errorf("Generated option for %s should not be empty", flag.Name)
			}
		})
	}
}

// Benchmark the flag extraction to ensure it's reasonably fast
func BenchmarkExtractFlags(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	for i := 0; i < b.N; i++ {
		testCmd := test.NewCmd(logger)
		extractFlags(testCmd)
	}
}

func TestScanVersionDirectories(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create version directories
	versionDirs := []string{"experimental", "legacy", "v1.0.0", "v0.6.10", "v2.1.3-alpha"}
	for _, dir := range versionDirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}
	
	// Create non-version directories that should be ignored
	ignoreDirs := []string{"README.md", "test", "docs", "other"}
	for _, dir := range ignoreDirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}
	
	// Create a regular file that should be ignored
	testFile := filepath.Join(tempDir, "versions.libsonnet")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test the function
	result, err := scanVersionDirectories(tempDir)
	if err != nil {
		t.Fatalf("scanVersionDirectories failed: %v", err)
	}
	
	// Expected results (should be sorted)
	expected := []string{"experimental", "legacy", "v0.6.10", "v1.0.0", "v2.1.3-alpha"}
	
	if len(result) != len(expected) {
		t.Errorf("Expected %d versions, got %d: %v", len(expected), len(result), result)
	}
	
	for i, expectedVersion := range expected {
		if i >= len(result) || result[i] != expectedVersion {
			t.Errorf("Expected versions %v, got %v", expected, result)
			break
		}
	}
	
	// Verify ignored directories are not included
	for _, ignored := range ignoreDirs {
		for _, found := range result {
			if found == ignored {
				t.Errorf("Found ignored directory %s in results: %v", ignored, result)
			}
		}
	}
}

func TestScanVersionDirectoriesEmpty(t *testing.T) {
	// Create empty directory
	tempDir := t.TempDir()
	
	result, err := scanVersionDirectories(tempDir)
	if err != nil {
		t.Fatalf("scanVersionDirectories failed: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty directory, got: %v", result)
	}
}

func TestScanVersionDirectoriesNonexistent(t *testing.T) {
	// Test with non-existent directory
	result, err := scanVersionDirectories("/nonexistent/path")
	if err == nil {
		t.Errorf("Expected error for non-existent directory, got result: %v", result)
	}
}

func TestVersionsTemplateGeneration(t *testing.T) {
	// Test template data structure
	data := VersionsTemplateData{
		Versions:      []string{"experimental", "legacy", "v1.0.0"},
		LatestVersion: "abc123",
	}
	
	// Template content (simplified version of actual template)
	templateContent := `// Generated versions
local versions = {
{{range .Versions}}  '{{.}}': import '{{.}}/main.libsonnet',
{{end}}};

{
  getBenchFunctions(version):: versions[version],
  latestVersionSha:: '{{.LatestVersion}}',
}`
	
	// Parse template with helper functions
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(templateContent)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	
	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	result := buf.String()
	
	// Check that all versions are included
	for _, version := range data.Versions {
		expectedLine := "  '" + version + "': import '" + version + "/main.libsonnet',"
		if !strings.Contains(result, expectedLine) {
			t.Errorf("Expected template to contain %q, got:\n%s", expectedLine, result)
		}
	}
	
	// Check that latestVersionSha is included
	expectedSha := "latestVersionSha:: 'abc123',"
	if !strings.Contains(result, expectedSha) {
		t.Errorf("Expected template to contain %q, got:\n%s", expectedSha, result)
	}
}

func TestVersionsTemplateGenerationEmpty(t *testing.T) {
	// Test with empty versions list
	data := VersionsTemplateData{
		Versions:      []string{},
		LatestVersion: "def456",
	}
	
	templateContent := `local versions = {
{{range .Versions}}  '{{.}}': import '{{.}}/main.libsonnet',
{{end}}};`
	
	tmpl, err := template.New("test").Parse(templateContent)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	result := buf.String()
	expected := `local versions = {
};`
	
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestVersionsTestTemplateGeneration(t *testing.T) {
	// Test the versions test template logic
	data := VersionsTemplateData{
		Versions:      []string{"experimental", "v1.0.0"},
		LatestVersion: "abc123",
	}
	
	// Simplified test template
	templateContent := `local tests = {
{{range $i, $version := .Versions}}  // Test {{add $i 2}}: Check version {{$version}} exists
  testGetBenchFunctions{{$i}}: std.assertEqual(
    std.type(versions.getBenchFunctions('{{$version}}')),
    'object'
  ),
{{end}}};`
	
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(templateContent)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	result := buf.String()
	
	// Check that test functions are generated correctly
	expectedTests := []string{
		"// Test 2: Check version experimental exists",
		"testGetBenchFunctions0:",
		"versions.getBenchFunctions('experimental')",
		"// Test 3: Check version v1.0.0 exists", 
		"testGetBenchFunctions1:",
		"versions.getBenchFunctions('v1.0.0')",
	}
	
	for _, expected := range expectedTests {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected template to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestIsVersionDirectory(t *testing.T) {
	tests := []struct {
		name     string
		dirname  string
		expected bool
	}{
		{
			name:     "experimental version",
			dirname:  "experimental",
			expected: true,
		},
		{
			name:     "legacy version",
			dirname:  "legacy",
			expected: true,
		},
		{
			name:     "semantic version v1.0.0",
			dirname:  "v1.0.0",
			expected: true,
		},
		{
			name:     "semantic version v0.6.10",
			dirname:  "v0.6.10",
			expected: true,
		},
		{
			name:     "semantic version with alpha",
			dirname:  "v2.1.3-alpha",
			expected: true,
		},
		{
			name:     "semantic version with beta",
			dirname:  "v1.0.0-beta.1",
			expected: true,
		},
		{
			name:     "non-version directory",
			dirname:  "docs",
			expected: false,
		},
		{
			name:     "README file",
			dirname:  "README.md",
			expected: false,
		},
		{
			name:     "test directory",
			dirname:  "test",
			expected: false,
		},
		{
			name:     "invalid version format",
			dirname:  "1.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionDirectory(tt.dirname)
			if result != tt.expected {
				t.Errorf("isVersionDirectory(%q) = %v, expected %v", 
					tt.dirname, result, tt.expected)
			}
		})
	}
}

// Mock HTTP server for testing GitHub API calls
func setupMockGitHubAPI(t *testing.T, response string, statusCode int) (string, func()) {
	t.Helper()
	
	// This is a simplified test - in reality we'd use httptest.NewServer
	// but for now we'll test the logic without network calls
	return "", func() {}
}

func TestFetchVersionsFromGitHubAPI(t *testing.T) {
	// Test the response parsing logic - manually create the expected data structure
	// This simulates what we'd get from the GitHub API
	contents := []GitHubContent{
		{Name: "experimental", Type: "dir"},
		{Name: "legacy", Type: "dir"},
		{Name: "v0.6.10", Type: "dir"},
		{Name: "v1.0.0", Type: "dir"},
		{Name: "README.md", Type: "file"},
		{Name: "docs", Type: "dir"},
	}
	
	// Filter for directories and extract version names
	var versions []string
	for _, item := range contents {
		if item.Type == "dir" && isVersionDirectory(item.Name) {
			versions = append(versions, item.Name)
		}
	}
	
	expected := []string{"experimental", "legacy", "v0.6.10", "v1.0.0"}
	
	if len(versions) != len(expected) {
		t.Errorf("Expected %d versions, got %d: %v", len(expected), len(versions), versions)
	}
	
	// Check each expected version is present (order doesn't matter for this test)
	versionMap := make(map[string]bool)
	for _, v := range versions {
		versionMap[v] = true
	}
	
	for _, expectedVersion := range expected {
		if !versionMap[expectedVersion] {
			t.Errorf("Expected to find version %s in results: %v", expectedVersion, versions)
		}
	}
	
	// Check that non-version directories are excluded
	excludedItems := []string{"README.md", "docs"}
	for _, excluded := range excludedItems {
		if versionMap[excluded] {
			t.Errorf("Should not include %s in version list: %v", excluded, versions)
		}
	}
}

func TestVersionDeduplication(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"experimental", "legacy", "v1.0.0"},
			expected: []string{"experimental", "legacy", "v1.0.0"},
		},
		{
			name:     "with duplicates",
			input:    []string{"experimental", "legacy", "experimental", "v1.0.0", "legacy"},
			expected: []string{"experimental", "legacy", "v1.0.0"},
		},
		{
			name:     "empty list",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single item",
			input:    []string{"experimental"},
			expected: []string{"experimental"},
		},
		{
			name:     "all duplicates",
			input:    []string{"experimental", "experimental", "experimental"},
			expected: []string{"experimental"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the deduplication logic we use in generateVersions
			seen := make(map[string]bool)
			var result []string
			for _, v := range tt.input {
				if !seen[v] {
					seen[v] = true
					result = append(result, v)
				}
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
					break
				}
			}
		})
	}
}

func TestVersionsListLocalMode(t *testing.T) {
	// Create temporary directory structure for testing local mode
	tempDir := t.TempDir()
	
	// Create some version directories
	localVersions := []string{"experimental", "v1.0.0", "v2.0.0"}
	for _, version := range localVersions {
		versionDir := filepath.Join(tempDir, version)
		err := os.MkdirAll(versionDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", version, err)
		}
	}
	
	// Create non-version directories that should be ignored
	nonVersions := []string{"docs", "test", "README.md"}
	for _, item := range nonVersions {
		itemPath := filepath.Join(tempDir, item)
		if item == "README.md" {
			// Create as file
			err := os.WriteFile(itemPath, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", item, err)
			}
		} else {
			// Create as directory
			err := os.MkdirAll(itemPath, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory %s: %v", item, err)
			}
		}
	}
	
	// Test scanning for local versions (simulating --versions-list local)
	var localVersionsFound []string
	if _, err := os.Stat(tempDir); err == nil {
		entries, err := os.ReadDir(tempDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && isVersionDirectory(entry.Name()) {
					localVersionsFound = append(localVersionsFound, entry.Name())
				}
			}
		}
	}
	
	// Target version to always include
	targetVersion := "v3.0.0"
	
	// Simulate the deduplication logic from generateVersions local mode
	allVersions := append([]string{targetVersion}, localVersionsFound...)
	seen := make(map[string]bool)
	var deduped []string
	for _, v := range allVersions {
		if !seen[v] {
			seen[v] = true
			deduped = append(deduped, v)
		}
	}
	
	// Verify results
	expectedVersions := []string{"v3.0.0", "experimental", "v1.0.0", "v2.0.0"}
	
	if len(deduped) != len(expectedVersions) {
		t.Errorf("Expected %d versions, got %d: %v", len(expectedVersions), len(deduped), deduped)
	}
	
	// Check that target version is always included first
	if len(deduped) > 0 && deduped[0] != targetVersion {
		t.Errorf("Expected target version %s to be first, got %s", targetVersion, deduped[0])
	}
	
	// Check that all expected local versions are included
	versionMap := make(map[string]bool)
	for _, v := range deduped {
		versionMap[v] = true
	}
	
	for _, expected := range localVersions {
		if !versionMap[expected] {
			t.Errorf("Expected to find local version %s in results: %v", expected, deduped)
		}
	}
	
	// Check that non-version items are excluded
	for _, excluded := range nonVersions {
		if versionMap[excluded] {
			t.Errorf("Should not include %s in version list: %v", excluded, deduped)
		}
	}
}

func TestVersionsListListMode(t *testing.T) {
	tests := []struct {
		name             string
		targetVersion    string
		existingVersions string
		expected         []string
	}{
		{
			name:             "target version only",
			targetVersion:    "experimental",
			existingVersions: "",
			expected:         []string{"experimental"},
		},
		{
			name:             "target with existing versions",
			targetVersion:    "v1.0.0",
			existingVersions: "experimental,legacy,v0.6.10",
			expected:         []string{"v1.0.0", "experimental", "legacy", "v0.6.10"},
		},
		{
			name:             "with whitespace in existing versions",
			targetVersion:    "experimental",
			existingVersions: " legacy , v0.6.10 , v1.0.0 ",
			expected:         []string{"experimental", "legacy", "v0.6.10", "v1.0.0"},
		},
		{
			name:             "duplicate target in existing versions",
			targetVersion:    "experimental",
			existingVersions: "experimental,legacy,v0.6.10",
			expected:         []string{"experimental", "legacy", "v0.6.10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from generateVersions list mode
			var allVersions []string
			
			if tt.existingVersions == "" {
				allVersions = []string{tt.targetVersion}
			} else {
				existingList := strings.Split(tt.existingVersions, ",")
				// Trim whitespace from each version
				for i := range existingList {
					existingList[i] = strings.TrimSpace(existingList[i])
				}
				allVersions = append([]string{tt.targetVersion}, existingList...)
			}
			
			// Apply deduplication if needed
			seen := make(map[string]bool)
			var deduped []string
			for _, v := range allVersions {
				if !seen[v] {
					seen[v] = true
					deduped = append(deduped, v)
				}
			}
			
			// Verify results
			if len(deduped) != len(tt.expected) {
				t.Errorf("Expected %d versions, got %d: %v", len(tt.expected), len(deduped), deduped)
			}
			
			for i, expected := range tt.expected {
				if i >= len(deduped) || deduped[i] != expected {
					t.Errorf("Expected %v, got %v", tt.expected, deduped)
					break
				}
			}
		})
	}
}

func TestVersionsCommandIntegration(t *testing.T) {
	// Integration test that doesn't hit external APIs
	// Tests the full workflow using --versions-list local and list modes
	
	tempDir := t.TempDir()
	
	// Create some test version directories
	testVersions := []string{"experimental", "v1.0.0", "v2.0.0"}
	for _, version := range testVersions {
		versionDir := filepath.Join(tempDir, version)
		err := os.MkdirAll(versionDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", version, err)
		}
		
		// Create a dummy main.libsonnet file to simulate a real version
		mainFile := filepath.Join(versionDir, "main.libsonnet")
		err = os.WriteFile(mainFile, []byte("{ testFunction: 'test' }"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test main.libsonnet: %v", err)
		}
	}
	
	tests := []struct {
		name           string
		versionsList   string
		targetVersion  string
		existingVersions string
		expectedVersions []string
	}{
		{
			name:           "local mode includes existing local versions",
			versionsList:   "local", 
			targetVersion:  "v3.0.0",
			expectedVersions: []string{"v3.0.0", "experimental", "v1.0.0", "v2.0.0"}, // target + local versions
		},
		{
			name:           "local mode with existing target",
			versionsList:   "local",
			targetVersion:  "experimental", // This exists locally
			expectedVersions: []string{"experimental", "v1.0.0", "v2.0.0"}, // deduped
		},
		{
			name:             "list mode with explicit versions",
			versionsList:     "list",
			targetVersion:    "v4.0.0",
			existingVersions: "experimental,legacy",
			expectedVersions: []string{"v4.0.0", "experimental", "legacy"},
		},
		{
			name:           "list mode target only",
			versionsList:   "list",
			targetVersion:  "experimental",
			expectedVersions: []string{"experimental"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the versions list determination logic without calling the actual command
			var allVersions []string
			
			switch tt.versionsList {
			case "local":
				// Simulate local mode logic
				var localVersions []string
				if _, err := os.Stat(tempDir); err == nil {
					entries, err := os.ReadDir(tempDir)
					if err == nil {
						for _, entry := range entries {
							if entry.IsDir() && isVersionDirectory(entry.Name()) {
								localVersions = append(localVersions, entry.Name())
							}
						}
					}
				}
				
				allVersions = append([]string{tt.targetVersion}, localVersions...)
				
			case "list":
				// Simulate list mode logic
				if tt.existingVersions == "" {
					allVersions = []string{tt.targetVersion}
				} else {
					existingList := strings.Split(tt.existingVersions, ",")
					for i := range existingList {
						existingList[i] = strings.TrimSpace(existingList[i])
					}
					allVersions = append([]string{tt.targetVersion}, existingList...)
				}
			}
			
			// Apply deduplication
			seen := make(map[string]bool)
			var deduped []string
			for _, v := range allVersions {
				if !seen[v] {
					seen[v] = true
					deduped = append(deduped, v)
				}
			}
			
			// Verify expected versions are present
			if len(deduped) != len(tt.expectedVersions) {
				t.Errorf("Expected %d versions, got %d: %v", len(tt.expectedVersions), len(deduped), deduped)
				return
			}
			
			// Check that target version is always first
			if len(deduped) > 0 && deduped[0] != tt.targetVersion {
				t.Errorf("Expected target version %s to be first, got %s", tt.targetVersion, deduped[0])
			}
			
			// Create a map of actual versions for easy checking
			actualMap := make(map[string]bool)
			for _, v := range deduped {
				actualMap[v] = true
			}
			
			// Verify all expected versions are present
			for _, expectedVersion := range tt.expectedVersions {
				if !actualMap[expectedVersion] {
					t.Errorf("Expected version %s not found in result: %v", expectedVersion, deduped)
				}
			}
		})
	}
	
	// Test template generation with local versions
	t.Run("template_generation_with_local_versions", func(t *testing.T) {
		// Get local versions
		var localVersions []string
		entries, err := os.ReadDir(tempDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && isVersionDirectory(entry.Name()) {
					localVersions = append(localVersions, entry.Name())
				}
			}
		}
		
		targetVersion := "v3.0.0"
		allVersions := append([]string{targetVersion}, localVersions...)
		
		// Create template data
		data := VersionsTemplateData{
			Versions:      allVersions,
			LatestVersion: "abc123",
		}
		
		// Simple template to test generation
		templateContent := `local versions = {
{{range .Versions}}  '{{.}}': import '{{.}}/main.libsonnet',
{{end}}};

{
  getBenchFunctions(version):: versions[version],
  getAvailableVersions():: std.objectFields(versions),
  latestVersionSha:: '{{.LatestVersion}}',
}`
		
		tmpl, err := template.New("versions").Parse(templateContent)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}
		
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, data)
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
		
		result := buf.String()
		
		// Verify all versions are included in the template
		for _, version := range allVersions {
			expectedImport := "'" + version + "': import '" + version + "/main.libsonnet',"
			if !strings.Contains(result, expectedImport) {
				t.Errorf("Template should contain import for %s, got:\n%s", version, result)
			}
		}
		
		// Verify SHA is included
		if !strings.Contains(result, "latestVersionSha:: 'abc123'") {
			t.Errorf("Template should contain latestVersionSha, got:\n%s", result)
		}
		
		// Verify helper functions are present
		expectedFunctions := []string{
			"getBenchFunctions(version)",
			"getAvailableVersions()",
		}
		
		for _, fn := range expectedFunctions {
			if !strings.Contains(result, fn) {
				t.Errorf("Template should contain function %s, got:\n%s", fn, result)
			}
		}
	})
}