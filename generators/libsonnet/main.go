package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/grafana/grafana-bench/cmd/test"
	"github.com/grafana/grafana-bench/generators/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const libsonnetTemplate = `local argo_workflows = (import 'argo-workflows-libsonnet/main.libsonnet').workflow.v1alpha1;
local base = (import './_base.libsonnet');
local templates = (import '../utils/templates.libsonnet');
local url = import 'github.com/jsonnet-libs/xtd/url.libsonnet';
local benchImage = '{{.BaseImageURL}}';
local benchPlaywrightImage = '{{.PlaywrightImageURL}}';

// Generated from bench {{.Version}} CLI flags
function(name) base(name, templates.bench_test.name) {
  local this = self,
  envVars:: [],

  parameters:: {
    container_patch: std.manifestJsonMinified({
      containers: [{
        name: 'main',
        env: this.envVars,
      }],
    }),
  },

  withEnvVars(envVars):: self {
    envVars+: envVars,
  },

  withContainerImage(image):: self {
    image: image,
  },

  withBenchTest(grafanaURL, suite):: self {
    // Default suite options derived from CLI flags
    local bench_options = {
      // bench image revision - pinned to this version
      benchRevision: '{{.Version}}',
{{.SuiteOptions}}
      // suite execution options
      options: {
        // prevent the workflow step to fail if the test suite fails
        noFail: false,
      },
      // environment variables to be passed to the test step in the workflow
      envVars: [],
    } + suite,

    // set step's env vars for test
    envVars+: bench_options.envVars,

    // create list of env=$env
    local test_env_vars = [env + '="$"' + env for env in bench_options.testEnvVars],

    local script = [
{{.ScriptFlags}}

    // Helper function to select the appropriate image based on test runner
    local selectedImage =
      if bench_options.testRunner == 'playwright' then
        benchPlaywrightImage
      else
        benchImage,

    parameters+: {
      script: std.join(' ', script),
      image: selectedImage + ':' + bench_options.benchRevision,
    },
  },
}
`

type FlagInfo struct {
	Name         string
	DefaultValue string
	Usage        string
	Type         string
	Deprecated   bool
}

type TemplateData struct {
	Version            string
	SuiteOptions       string
	ScriptFlags        string
	BaseImageURL       string
	PlaywrightImageURL string
}

func main() {
	var outputPath string
	var version string
	var baseImageURL string
	var playwrightImageURL string
	
	flag.StringVar(&outputPath, "o", "", "output directory for generated libsonnet")
	flag.StringVar(&version, "version", "", "version to pin in the generated library")
	flag.StringVar(&baseImageURL, "base-image", "us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench", "base bench image URL")
	flag.StringVar(&playwrightImageURL, "playwright-image", "us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench-playwright", "playwright bench image URL")
	flag.Parse()

	if outputPath == "" {
		log.Fatal("output directory required (-o)")
	}

	if version == "" {
		// Get the latest git tag - same approach as gendoc
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}

		version, err = utils.GetLatestBenchTag(workDir)
		if err != nil {
			log.Fatalf("Failed to get latest bench tag: %v", err)
		}
		
		fmt.Printf("Using latest tag: %s\n", version)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	// Create test command to extract flags
	testCmd := test.NewCmd(logger)
	
	// Extract flags information
	flags := extractFlags(testCmd)
	
	fmt.Printf("Extracted %d non-deprecated flags\n", len(flags))
	
	// Generate template data
	data := TemplateData{
		Version:            version,
		SuiteOptions:       generateSuiteOptions(flags),
		ScriptFlags:        generateScriptFlags(flags),
		BaseImageURL:       baseImageURL,
		PlaywrightImageURL: playwrightImageURL,
	}
	
	// Generate libsonnet
	tmpl, err := template.New("libsonnet").Parse(libsonnetTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	
	// Ensure output directory exists
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Write libsonnet file
	outputFile := filepath.Join(outputPath, "bench.libsonnet")
	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()
	
	err = tmpl.Execute(file, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	file.Close()
	
	// Format the generated libsonnet with jsonnetfmt
	err = formatLibsonnet(outputFile)
	if err != nil {
		log.Fatalf("Failed to format libsonnet: %v", err)
	}
	
	fmt.Printf("Generated libsonnet for version %s at %s\n", version, outputFile)
}

func extractFlags(cmd *cobra.Command) []FlagInfo {
	var flags []FlagInfo
	
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		// Skip flags that don't belong in the libsonnet
		if shouldSkipFlag(flag.Name, flag.Usage) {
			return
		}
		
		deprecated := strings.Contains(flag.Usage, "deprecated")
		
		flags = append(flags, FlagInfo{
			Name:         flag.Name,
			DefaultValue: flag.DefValue,
			Usage:        flag.Usage,
			Type:         flag.Value.Type(),
			Deprecated:   deprecated,
		})
	})
	
	// Sort flags by name for consistent output
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Name < flags[j].Name
	})
	
	return flags
}

func shouldSkipFlag(name, usage string) bool {
	// Skip meta flags and internal flags
	skipFlags := map[string]bool{
		"help":           true,
		"config":         true,
		"bench-revision": true, // We handle this specially
	}
	
	if skipFlags[name] {
		return true
	}
	
	// Skip deprecated flags (check both usage text and if name contains deprecated patterns)
	if strings.Contains(usage, "deprecated") {
		return true
	}
	
	// Also skip flags that are clearly deprecated variants based on current libsonnet
	deprecatedFlags := map[string]bool{
		"test-suite":              true, // old form of suite-path
		"test-suite-base":         true, // old form of suite-base
		"test-suite-name":         true, // old form of suite-name
		"test-suite-repo":         true, // old form of suite-repo-url
		"test-suite-repo-dirs":    true, // old form of suite-repo-dirs
		"test-suite-repo-token":   true, // old form of suite-repo-token
		"test-suite-revision":     true, // old form of suite-revision
		"test-env-vars":           true, // old form of test-env
		"pw-execute-cmd":          true, // old form of pw-execute
		"pw-prepare-cmd":          true, // old form of pw-prepare
		"codeowners-mapping":      true, // old form of slack-codeowners-mapping
		"dashboard":               true, // old form of run-dashboard
		"format":                  true, // old form of report-output
		"notify-passing":          true, // old form of slack-passing
		"run-trigger":             true, // old form of run-stage
		"test-trigger":            true, // old form of run-stage
		"trigger":                 true, // old form of run-stage
		"verbose":                 true, // old form of test-verbose
		"k6-cloud-project-id":     true, // old form of k6-cloud-project
		"report-format":           true, // old form of report-output
		"test-report-format":      true, // old form of report-output
		"suite-run-metrics":       true, // old form of run-metrics
		"suite-run-metrics-prefix": true, // old form of run-metrics-prefix
	}
	
	return deprecatedFlags[name]
}

func generateSuiteOptions(flags []FlagInfo) string {
	var options []string
	
	for _, flag := range flags {
		if flag.Deprecated {
			continue
		}
		
		// Convert CLI flag to libsonnet option
		libsonnetOption := generateLibsonnetOption(flag)
		if libsonnetOption != "" {
			options = append(options, "      "+libsonnetOption+",")
		}
	}
	
	return strings.Join(options, "\n")
}

func generateLibsonnetOption(flag FlagInfo) string {
	// Convert kebab-case to camelCase for libsonnet
	camelCase := toCamelCase(flag.Name)
	
	// Determine default value based on flag type and current default
	defaultValue := getLibsonnetDefaultValue(flag)
	
	// Add comment with original flag name and description
	comment := fmt.Sprintf("// --%s: %s", flag.Name, cleanUsage(flag.Usage))
	
	return fmt.Sprintf("%s\n      %s: %s", comment, camelCase, defaultValue)
}

func generateScriptFlags(flags []FlagInfo) string {
	var baseArrayParts []string
	var concatenationParts []string
	
	// Always include basic required flags in the base array
	baseArrayParts = append(baseArrayParts, "                     'grafana-bench',")
	baseArrayParts = append(baseArrayParts, "                     'test',")
	
	// Add required string flags to base array
	baseArrayParts = append(baseArrayParts, "                     '--grafana-url',")
	baseArrayParts = append(baseArrayParts, "                     grafanaURL,")
	baseArrayParts = append(baseArrayParts, "                     '--suite-path',")
	baseArrayParts = append(baseArrayParts, "                     bench_options.path,")

	// Process flags for optional concatenations only
	for _, flag := range flags {
		if flag.Deprecated {
			continue
		}
		
		// Skip required flags that are already in the base array
		if isRequiredStringFlag(flag.Name) {
			continue
		}
		
		// Generate optional script flag for concatenation
		scriptFlag := generateOptionalScriptFlag(flag)
		if scriptFlag != "" {
			concatenationParts = append(concatenationParts, scriptFlag)
		}
	}
	
	// Always include these required flags in base array
	baseArrayParts = append(baseArrayParts, "                     '--log-level',")
	baseArrayParts = append(baseArrayParts, "                     'info',")
	baseArrayParts = append(baseArrayParts, "                     '--test-report-format',")
	baseArrayParts = append(baseArrayParts, "                     'log',")
	
	// Join base array
	baseArray := strings.Join(baseArrayParts, "\n")
	
	// Join with concatenations and add the final noFail option
	result := baseArray + "\n                   ]"
	if len(concatenationParts) > 0 {
		result += " " + strings.Join(concatenationParts, "\n")
	}
	result += "\n                   + ([if bench_options.options.noFail then '|| true']),  // MUST be the last option"
	
	return result
}

func generateOptionalScriptFlag(flag FlagInfo) string {
	camelCase := toCamelCase(flag.Name)
	
	switch flag.Type {
	case "bool":
		return fmt.Sprintf("+ (if bench_options.%s then ['--%s'] else [])", camelCase, flag.Name)
	case "string":
		// Skip required strings - they're handled in the base array
		if isRequiredStringFlag(flag.Name) {
			return ""
		}
		// Handle special string escaping cases
		if strings.Contains(flag.Name, "pw-") {
			return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', std.escapeStringBash(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
		}
		if flag.Name == "run-dashboard" {
			return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', url.escapeString(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
		}
		return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', bench_options.%s] else [])", camelCase, flag.Name, camelCase)
	case "stringArray", "stringSlice":
		// Handle special cases
		if flag.Name == "suite-repo-dirs" {
			return fmt.Sprintf("+ std.flattenArrays([['--%s', dir] for dir in bench_options.%s])", flag.Name, camelCase)
		}
		return fmt.Sprintf("+ std.flattenArrays([['--%s', item] for item in bench_options.%s])", flag.Name, camelCase)
	case "stringToString":
		// Special handling for test-env which becomes test-env-vars in the CLI but testEnvVars in libsonnet
		if flag.Name == "test-env" {
			return "+ (if test_env_vars != [] then ['--test-env', std.join(',', test_env_vars)] else [])"
		}
		return fmt.Sprintf("+ (if std.length(bench_options.%s) > 0 then ['--%s', std.join(',', [k + '=' + v for k, v in std.objectKeysValues(bench_options.%s)])] else [])", camelCase, flag.Name, camelCase)
	case "int":
		return fmt.Sprintf("+ (if bench_options.%s != 0 then ['--%s', std.toString(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
	case "duration":
		return fmt.Sprintf("+ (if bench_options.%s != '0s' then ['--%s', bench_options.%s] else [])", camelCase, flag.Name, camelCase)
	default:
		// For other types, treat as string
		return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', std.toString(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
	}
}

func toCamelCase(kebab string) string {
	parts := strings.Split(kebab, "-")
	if len(parts) == 1 {
		return parts[0]
	}
	
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}

func getLibsonnetDefaultValue(flag FlagInfo) string {
	switch flag.Type {
	case "bool":
		if flag.DefaultValue == "true" {
			return "true"
		}
		return "false"
	case "string":
		if flag.DefaultValue == "" {
			if isRequiredStringFlag(flag.Name) {
				return fmt.Sprintf("error 'must define %s'", strings.ReplaceAll(flag.Name, "-", " "))
			}
			return "''"
		}
		return fmt.Sprintf("'%s'", flag.DefaultValue)
	case "stringArray", "stringSlice":
		return "[]"
	case "stringToString":
		// Special case for test-env which becomes testEnvVars in the libsonnet
		if flag.Name == "test-env" {
			return "[]"
		}
		return "{}"
	case "int":
		if flag.DefaultValue == "" {
			return "0"
		}
		return flag.DefaultValue
	case "duration":
		if flag.DefaultValue == "" || flag.DefaultValue == "0s" {
			return "'0s'"
		}
		return fmt.Sprintf("'%s'", flag.DefaultValue)
	default:
		if flag.DefaultValue == "" {
			return "''"
		}
		return fmt.Sprintf("'%s'", flag.DefaultValue)
	}
}

func isRequiredStringFlag(flagName string) bool {
	// These flags are required in the original libsonnet (have error defaults)
	requiredFlags := map[string]bool{
		"grafana-url": true,
		"suite-path":  true,
	}
	return requiredFlags[flagName]
}

func cleanUsage(usage string) string {
	// Remove newlines and extra whitespace for comments
	cleaned := strings.ReplaceAll(usage, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	// Remove multiple spaces
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}
	return strings.TrimSpace(cleaned)
}

// formatLibsonnet formats the generated libsonnet file using jsonnetfmt
func formatLibsonnet(filename string) error {
	// Check if jsonnetfmt is available
	_, err := exec.LookPath("jsonnetfmt")
	if err != nil {
		return fmt.Errorf("jsonnetfmt not found: %w\n\nTo install jsonnetfmt, run: make install-deps", err)
	}
	
	// Use jsonnetfmt -i to format in place, same as deployment_tools
	cmd := exec.Command("jsonnetfmt", "-i", filename)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("formatting libsonnet with jsonnetfmt: %w\nOutput: %s", err, output)
	}
	
	fmt.Printf("Formatted libsonnet with jsonnetfmt\n")
	return nil
}