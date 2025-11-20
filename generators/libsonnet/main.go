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

type VersionsTemplateData struct {
	Versions []string
}

func main() {
	// Check if we're being called with subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "generate":
			generateMain()
			return
		case "versions":
			generateVersions()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	// Default behavior (for backward compatibility)
	generateMain()
}

func printHelp() {
	fmt.Println("Grafana Bench Libsonnet Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run generators/libsonnet generate [flags]    Generate main.libsonnet and main_test.jsonnet")
	fmt.Println("  go run generators/libsonnet versions [flags]    Generate versions.libsonnet")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  generate    Generate main libsonnet functions for a specific version")
	fmt.Println("  versions    Generate versions mapping libsonnet from a list of versions")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run generators/libsonnet generate -o libsonnet -version v1.0.0")
	fmt.Println("  go run generators/libsonnet versions --versions \"experimental,v1.0.0,v1.1.0\" -o libsonnet")
}

func generateMain() {
	var outputPath string
	var version string
	var baseImageURL string
	var playwrightImageURL string
	
	// Parse args starting from index 2 if first arg is "generate"
	var args []string
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		args = os.Args[2:]
	} else {
		args = os.Args[1:]
	}
	
	// Create a new FlagSet for parsing
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.StringVar(&outputPath, "o", "", "output directory for generated libsonnet")
	fs.StringVar(&version, "version", "", "version to pin in the generated library")
	fs.StringVar(&baseImageURL, "base-image", "us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench", "base bench image URL")
	fs.StringVar(&playwrightImageURL, "playwright-image", "us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench-playwright", "playwright bench image URL")
	fs.Parse(args)

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
			// Fallback to "dev-{shortSHA}" if no tags found - this matches dev image tagging
			shortSHA, shaErr := utils.GetShortCommitSHA(workDir)
			if shaErr != nil {
				version = "dev-latest"
				fmt.Printf("No git tags found and unable to get commit SHA, using fallback version: %s\n", version)
			} else {
				version = fmt.Sprintf("dev-%s", shortSHA)
				fmt.Printf("No git tags found, using fallback version: %s\n", version)
			}
		} else {
			fmt.Printf("Using latest tag: %s\n", version)
		}
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
	
	// Load templates from files
	mainTemplate, err := loadTemplate("main.libsonnet.tmpl")
	if err != nil {
		log.Fatalf("Failed to load main template: %v", err)
	}
	
	testTemplate, err := loadTemplate("main_test.jsonnet.tmpl") 
	if err != nil {
		log.Fatalf("Failed to load test template: %v", err)
	}
	
	// Parse templates
	mainTmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		log.Fatalf("Failed to parse main template: %v", err)
	}
	
	testTmpl, err := template.New("test").Parse(testTemplate)
	if err != nil {
		log.Fatalf("Failed to parse test template: %v", err)
	}
	
	// Ensure output directory exists
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Write main libsonnet file
	mainFile := filepath.Join(outputPath, "main.libsonnet")
	file, err := os.Create(mainFile)
	if err != nil {
		log.Fatalf("Failed to create main file: %v", err)
	}
	defer file.Close()
	
	err = mainTmpl.Execute(file, data)
	if err != nil {
		log.Fatalf("Failed to execute main template: %v", err)
	}
	file.Close()
	
	// Write test file
	testFile := filepath.Join(outputPath, "main_test.jsonnet")
	testFileHandle, err := os.Create(testFile)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	defer testFileHandle.Close()
	
	err = testTmpl.Execute(testFileHandle, data)
	if err != nil {
		log.Fatalf("Failed to execute test template: %v", err)
	}
	testFileHandle.Close()
	
	// Format the generated files with jsonnetfmt
	err = formatLibsonnet(mainFile)
	if err != nil {
		log.Fatalf("Failed to format main libsonnet: %v", err)
	}
	
	err = formatLibsonnet(testFile)
	if err != nil {
		log.Fatalf("Failed to format test file: %v", err)
	}
	
	fmt.Printf("Generated main.libsonnet for version %s at %s\n", version, mainFile)
	fmt.Printf("Generated main_test.jsonnet at %s\n", testFile)
}

func generateVersions() {
	var outputPath string
	var versionsStr string
	
	// Parse args starting from index 2 since first arg is "versions"
	args := os.Args[2:]
	
	// Create a new FlagSet for parsing
	fs := flag.NewFlagSet("versions", flag.ExitOnError)
	fs.StringVar(&outputPath, "o", "", "output directory for generated libsonnet")
	fs.StringVar(&versionsStr, "versions", "", "comma-separated list of versions")
	fs.Parse(args)

	if outputPath == "" {
		log.Fatal("output directory required (-o)")
	}

	if versionsStr == "" {
		log.Fatal("versions list required (--versions)")
	}

	// Parse versions
	versions := strings.Split(versionsStr, ",")
	for i, v := range versions {
		versions[i] = strings.TrimSpace(v)
	}

	// Sort versions
	sort.Strings(versions)
	
	fmt.Printf("Generating versions.libsonnet with %d versions: %s\n", len(versions), strings.Join(versions, ", "))
	
	// Generate template data
	data := VersionsTemplateData{
		Versions: versions,
	}
	
	// Load template from file
	versionsTemplate, err := loadTemplate("versions.libsonnet.tmpl")
	if err != nil {
		log.Fatalf("Failed to load versions template: %v", err)
	}
	
	// Parse template
	versionsTmpl, err := template.New("versions").Parse(versionsTemplate)
	if err != nil {
		log.Fatalf("Failed to parse versions template: %v", err)
	}
	
	// Ensure output directory exists
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Write versions file
	versionsFile := filepath.Join(outputPath, "versions.libsonnet")
	file, err := os.Create(versionsFile)
	if err != nil {
		log.Fatalf("Failed to create versions file: %v", err)
	}
	defer file.Close()
	
	err = versionsTmpl.Execute(file, data)
	if err != nil {
		log.Fatalf("Failed to execute versions template: %v", err)
	}
	file.Close()
	
	// Generate versions test file
	versionsTestTemplate, err := loadTemplate("versions_test.jsonnet.tmpl")
	if err != nil {
		log.Fatalf("Failed to load versions test template: %v", err)
	}
	
	versionsTestTmpl, err := template.New("versions_test").Parse(versionsTestTemplate)
	if err != nil {
		log.Fatalf("Failed to parse versions test template: %v", err)
	}
	
	versionsTestFile := filepath.Join(outputPath, "versions_test.jsonnet")
	testFile, err := os.Create(versionsTestFile)
	if err != nil {
		log.Fatalf("Failed to create versions test file: %v", err)
	}
	defer testFile.Close()
	
	err = versionsTestTmpl.Execute(testFile, data)
	if err != nil {
		log.Fatalf("Failed to execute versions test template: %v", err)
	}
	testFile.Close()
	
	// Format the generated files with jsonnetfmt
	err = formatLibsonnet(versionsFile)
	if err != nil {
		log.Fatalf("Failed to format versions libsonnet: %v", err)
	}
	
	err = formatLibsonnet(versionsTestFile)
	if err != nil {
		log.Fatalf("Failed to format versions test file: %v", err)
	}
	
	fmt.Printf("Generated versions.libsonnet at %s\n", versionsFile)
	fmt.Printf("Generated versions_test.jsonnet at %s\n", versionsTestFile)
}

func loadTemplate(filename string) (string, error) {
	templatePath := filepath.Join("generators", "libsonnet", "tmpl", filename)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}
	return string(content), nil
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
	
	// Skip go-test related flags since they're not relevant for libsonnet
	goTestFlags := map[string]bool{
		"go-args":         true, // arguments to go test command - not needed for libsonnet
		"go-retries":      true, // number of retries for failed go tests - not needed for libsonnet
		"go-test-args":    true, // arguments to go test using arg flag - not needed for libsonnet  
		"go-test-packages": true, // packages for go testing - not needed for libsonnet
	}
	
	return deprecatedFlags[name] || goTestFlags[name]
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
	baseArrayParts = append(baseArrayParts, "                     bench_options.suitePath,")

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
			return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', escapeString(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
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