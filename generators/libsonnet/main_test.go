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
			expected: "true",
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
			expected: "'default-value'",
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
			expected: "3",
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
			expected: "// --test-type: test type. Allowed values: 'smoke', 'load'\n      testType: 'smoke'",
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