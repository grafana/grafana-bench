package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ArgoWorkflowStep represents the structure we expect from the jsonnet output
type ArgoWorkflowStep struct {
	Parameters struct {
		Script string `json:"script"`
		Image  string `json:"image"`
	} `json:"parameters"`
}

// TestCase represents a test configuration
type TestCase struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	GrafanaURL     string                 `json:"grafanaURL"`
	Suite          map[string]interface{} `json:"suite"`
	ExpectedFlags  []string               `json:"expectedFlags"`
	ForbiddenFlags []string               `json:"forbiddenFlags,omitempty"`
}

func TestLibsonnetConversion(t *testing.T) {
	// Check if jsonnet binary is available
	if _, err := exec.LookPath("jsonnet"); err != nil {
		t.Fatalf("jsonnet binary not found: %v", err)
	}

	// Create temporary directory for generated libsonnet
	tempDir, err := os.MkdirTemp("", "libsonnet_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate libsonnet in temp directory
	libsonnetPath := filepath.Join(tempDir, "bench.libsonnet")
	if err := generateLibsonnet(libsonnetPath); err != nil {
		t.Fatalf("Failed to generate libsonnet: %v", err)
	}

	// Load test cases
	testCases := loadTestCases(t)

	// Run each test case
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if err := runTestCase(t, testCase, libsonnetPath); err != nil {
				t.Errorf("Test case failed: %v", err)
			}
		})
	}
}

func generateLibsonnet(outputPath string) error {
	// Run the main libsonnet generator with custom output path and test version
	outputDir := filepath.Dir(outputPath)
	cmd := exec.Command("go", "run", "main.go", "-o", outputDir, "-version", "v1.0.0-test")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libsonnet generation failed: %v\nOutput: %s", err, string(output))
	}
	
	return nil
}

func loadTestCases(t *testing.T) []TestCase {
	// For now, return hardcoded test cases
	// TODO: Load from version-specific JSON files in testdata/
	return []TestCase{
		{
			Name:       "basic-k6-test",
			Description: "Basic K6 smoke test configuration",
			GrafanaURL: "http://localhost:3000",
			Suite: map[string]interface{}{
				"testRunner": "k6",
				"path":       "grafana-core",
				"type":       "smoke",
			},
			ExpectedFlags: []string{
				"grafana-bench", "test",
				"--grafana-url", "http://localhost:3000",
				"--suite-path", "grafana-core",
				"--test-runner", "k6",
				"--test-type", "smoke",
			},
		},
		{
			Name:       "playwright-test-with-metrics",
			Description: "Playwright test with prometheus metrics enabled",
			GrafanaURL: "http://localhost:3000",
			Suite: map[string]interface{}{
				"testRunner":        "playwright",
				"path":              "browser-tests",
				"type":              "smoke",
				"prometheusMetrics": true,
				"pwPrepare":         "yarn install; yarn playwright install chromium",
				"pwExecute":         "yarn playwright test",
			},
			ExpectedFlags: []string{
				"--test-runner", "playwright",
				"--suite-path", "browser-tests",
				"--prometheus-metrics",
				"--pw-prepare", "yarn install; yarn playwright install chromium",
				"--pw-execute", "yarn playwright test",
			},
		},
		{
			Name:       "k6-with-slack-notifications",
			Description: "K6 test with Slack notifications and custom attributes",
			GrafanaURL: "http://localhost:3000",
			Suite: map[string]interface{}{
				"testRunner":         "k6",
				"path":               "api-tests",
				"slackNotifications": true,
				"slackPassing":       true,
				"runAttribute":       []string{"environment=test", "team=backend"},
			},
			ExpectedFlags: []string{
				"--slack-notifications",
				"--slack-passing",
				"--run-attribute", "environment=test",
				"--run-attribute", "team=backend",
			},
		},
		{
			Name:       "version-specific-image-selection",
			Description: "Test that correct image is selected based on test runner",
			GrafanaURL: "http://localhost:3000",
			Suite: map[string]interface{}{
				"testRunner": "playwright",
				"path":       "browser-tests",
			},
			ExpectedFlags: []string{
				"--test-runner", "playwright",
			},
		},
	}
}

func runTestCase(t *testing.T, testCase TestCase, libsonnetPath string) error {
	// Read the generated libsonnet and rewrite import paths for testing
	content, err := os.ReadFile(libsonnetPath)
	if err != nil {
		return fmt.Errorf("failed to read generated libsonnet: %v", err)
	}
	
	// Rewrite import paths to use our local dependencies
	libsonnetContent := string(content)
	libsonnetContent = strings.ReplaceAll(libsonnetContent, "import './_base.libsonnet'", "import 'deps/_base.libsonnet'")
	libsonnetContent = strings.ReplaceAll(libsonnetContent, "import '../utils/templates.libsonnet'", "import 'deps/utils/templates.libsonnet'")
	libsonnetContent = strings.ReplaceAll(libsonnetContent, "import '../../infra-utils/version_comparisons.libsonnet'", "import 'deps/infra-utils/version_comparisons.libsonnet'")
	libsonnetContent = strings.ReplaceAll(libsonnetContent, "import 'argo-workflows-libsonnet/main.libsonnet'", "import 'deps/vendor/argo-workflows-libsonnet/main.libsonnet'")
	libsonnetContent = strings.ReplaceAll(libsonnetContent, "import 'github.com/jsonnet-libs/xtd/url.libsonnet'", "import 'deps/vendor/github.com/jsonnet-libs/xtd/url.libsonnet'")
	
	// Create jsonnet test snippet that uses the rewritten libsonnet
	suiteJSON, err := json.Marshal(testCase.Suite)
	if err != nil {
		return fmt.Errorf("failed to marshal suite config: %v", err)
	}

	// Write the rewritten libsonnet to a temp file and import it
	tempLibsonnet, err := os.CreateTemp("", "bench_*.libsonnet")
	if err != nil {
		return fmt.Errorf("failed to create temp libsonnet file: %v", err)
	}
	defer os.Remove(tempLibsonnet.Name())
	
	if _, err := tempLibsonnet.WriteString(libsonnetContent); err != nil {
		return fmt.Errorf("failed to write temp libsonnet: %v", err)
	}
	tempLibsonnet.Close()

	jsonnetCode := fmt.Sprintf(`
local bench = import '%s';
local result = bench('test-step').withBenchTest('%s', %s);
// Explicitly expose hidden fields for testing
{
  parameters: {
    script: result.parameters.script,
    image: result.parameters.image,
  }
}
`, tempLibsonnet.Name(), testCase.GrafanaURL, string(suiteJSON))

	// Debug: also log what we're trying to execute
	t.Logf("Generated jsonnet code: %s", jsonnetCode)

	// Write temporary jsonnet test file
	tempFile, err := os.CreateTemp("", "test_*.jsonnet")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(jsonnetCode); err != nil {
		return fmt.Errorf("failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Execute jsonnet with library path pointing to current directory (generators/libsonnet)
	cmd := exec.Command("jsonnet", "-J", ".", tempFile.Name())
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("jsonnet execution failed: %v\nstderr: %s", err, exitError.Stderr)
		}
		return fmt.Errorf("jsonnet execution failed: %v", err)
	}

	// Debug: log the raw output
	t.Logf("Raw jsonnet output: %s", string(output))

	// Parse output
	var result ArgoWorkflowStep
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse jsonnet output: %v\nOutput: %s", err, string(output))
	}

	// Validate expected flags are present
	script := result.Parameters.Script
	for _, expectedFlag := range testCase.ExpectedFlags {
		if !strings.Contains(script, expectedFlag) {
			return fmt.Errorf("expected flag '%s' not found in generated script: %s", expectedFlag, script)
		}
	}

	// Validate forbidden flags are not present
	for _, forbiddenFlag := range testCase.ForbiddenFlags {
		if strings.Contains(script, forbiddenFlag) {
			return fmt.Errorf("forbidden flag '%s' found in generated script: %s", forbiddenFlag, script)
		}
	}

	// Validate image selection
	if result.Parameters.Image == "" {
		return fmt.Errorf("no image specified in generated config")
	}

	// Additional validation for playwright image selection
	if testCase.Name == "version-specific-image-selection" {
		if !strings.Contains(result.Parameters.Image, "playwright") {
			return fmt.Errorf("expected playwright image for playwright test runner, got: %s", result.Parameters.Image)
		}
	}

	t.Logf("✅ %s - Generated script: %s", testCase.Description, script)
	return nil
}