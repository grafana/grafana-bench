package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

type GitHubContent struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type TemplateData struct {
	Version            string
	SuiteOptions       string
	ScriptFlags        string
	BaseImageURL       string
	PlaywrightImageURL string
	URLParamName       string // grafanaURL for v0.6.11, serviceURL for v1.0.0+
	URLFlagName        string // --grafana-url for v0.6.11, --service-url for v1.0.0+
}

type VersionsTemplateData struct {
	Versions      []string
	LatestVersion string
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
		case "fetchVersions":
			fetchVersionsMain()
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
	fmt.Println("  go run generators/libsonnet fetchVersions [flags] Discover available versions from deployment_tools repo")
	fmt.Println("  go run generators/libsonnet generate [flags]     Generate main.libsonnet and main_test.jsonnet")
	fmt.Println("  go run generators/libsonnet versions [flags]     Generate versions.libsonnet")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  fetchVersions    Discover available versions from deployment_tools repo")
	fmt.Println("  generate         Generate main libsonnet functions for a specific version")
	fmt.Println("  versions         Generate versions mapping libsonnet")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Discover versions (uses temp dir, default repo, GITHUB_TOKEN env var)")
	fmt.Println("  go run generators/libsonnet fetchVersions")
	fmt.Println()
	fmt.Println("  # Discover versions and checkout to directory")
	fmt.Println("  go run generators/libsonnet fetchVersions -o /tmp/work")
	fmt.Println()
	fmt.Println("  # Discover versions with explicit token")
	fmt.Println("  go run generators/libsonnet fetchVersions --github-token TOKEN -o /tmp/work")
	fmt.Println()
	fmt.Println("  # Generate version")
	fmt.Println("  go run generators/libsonnet generate --target-version experimental --latest-version-sha abc123 -o /tmp/work")
	fmt.Println()
	fmt.Println("  # Generate versions mapping (fetch from remote)")
	fmt.Println("  go run generators/libsonnet versions --target-version experimental --versions-list fetch --latest-version-sha abc123 -o /tmp/work")
	fmt.Println()
	fmt.Println("  # Generate versions mapping (local only - for testing)")
	fmt.Println("  go run generators/libsonnet versions --target-version experimental --versions-list local --latest-version-sha abc123 -o /tmp/work")
	fmt.Println()
	fmt.Println("  # Generate versions mapping (explicit list)")
	fmt.Println("  go run generators/libsonnet versions --target-version experimental --versions-list list --existing-versions \"legacy,v0.6.10\" --latest-version-sha abc123 -o /tmp/work")
}

func generateMain() {
	var outputPath string
	var targetVersion string

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
	fs.StringVar(&targetVersion, "target-version", "", "version to pin in the generated library")
	// Image URLs are auto-detected based on version (experimental = dev, releases = prod)
	fs.Parse(args)

	if outputPath == "" {
		log.Fatal("output directory required (-o)")
	}

	if targetVersion == "" {
		// Get the latest git tag - same approach as gendoc
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}

		targetVersion, err = utils.GetLatestBenchTag(workDir)
		if err != nil {
			// Fallback to "dev-{shortSHA}" if no tags found - this matches dev image tagging
			shortSHA, shaErr := utils.GetShortCommitSHA(workDir)
			if shaErr != nil {
				targetVersion = "dev-latest"
				fmt.Printf("No git tags found and unable to get commit SHA, using fallback version: %s\n", targetVersion)
			} else {
				targetVersion = fmt.Sprintf("dev-%s", shortSHA)
				fmt.Printf("No git tags found, using fallback version: %s\n", targetVersion)
			}
		} else {
			fmt.Printf("Using latest tag: %s\n", targetVersion)
		}
	}

	// Determine image URLs based on version
	var baseImageURL, playwrightImageURL string

	if targetVersion == "experimental" {
		// For experimental versions, use dev images with dev-{shortSha} tag
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}

		shortSHA, err := utils.GetShortCommitSHA(workDir)
		if err != nil {
			log.Fatalf("Failed to get commit SHA for experimental version: %v", err)
		}

		baseImageURL = fmt.Sprintf("us-docker.pkg.dev/grafanalabs-dev/docker-grafana-bench-dev/grafana-bench:dev-%s", shortSHA)
		playwrightImageURL = fmt.Sprintf("us-docker.pkg.dev/grafanalabs-dev/docker-grafana-bench-dev/grafana-bench-playwright:dev-%s", shortSHA)
	} else {
		// For release versions, use prod images with version tag
		baseImageURL = fmt.Sprintf("us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:%s", targetVersion)
		playwrightImageURL = fmt.Sprintf("us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench-playwright:%s", targetVersion)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create test command to extract flags
	testCmd := test.NewCmd(logger)

	// Extract flags information
	flags := extractFlags(testCmd)

	fmt.Printf("Extracted %d non-deprecated flags\n", len(flags))

	// Determine URL parameter name based on version
	urlParamName := "serviceURL"
	urlFlagName := "--service-url"
	if targetVersion == "v0.6.11" {
		urlParamName = "grafanaURL"
		urlFlagName = "--grafana-url"
	}

	// Generate template data
	data := TemplateData{
		Version:            targetVersion,
		SuiteOptions:       generateSuiteOptions(flags),
		ScriptFlags:        generateScriptFlags(flags, targetVersion),
		BaseImageURL:       baseImageURL,
		PlaywrightImageURL: playwrightImageURL,
		URLParamName:       urlParamName,
		URLFlagName:        urlFlagName,
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

	// Create version-specific subdirectory
	versionDir := filepath.Join(outputPath, targetVersion)
	err = os.MkdirAll(versionDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create version directory: %v", err)
	}

	// Write main libsonnet file in version subdirectory
	mainFile := filepath.Join(versionDir, "main.libsonnet")
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

	// Write test file in version subdirectory
	testFile := filepath.Join(versionDir, "main_test.jsonnet")
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

	fmt.Printf("Generated main.libsonnet for version %s at %s\n", targetVersion, mainFile)
	fmt.Printf("Generated main_test.jsonnet at %s\n", testFile)
}

func generateVersions() {
	var outputPath string
	var targetVersion string
	var existingVersions string
	var latestVersionSha string
	var versionsList string

	// Parse args starting from index 2 since first arg is "versions"
	args := os.Args[2:]

	// Create a new FlagSet for parsing
	fs := flag.NewFlagSet("versions", flag.ExitOnError)
	fs.StringVar(&outputPath, "o", "", "output directory for generated libsonnet")
	fs.StringVar(&targetVersion, "target-version", "", "version being created/updated (e.g., 'experimental', 'v1.2.3')")
	fs.StringVar(&existingVersions, "existing-versions", "", "comma-separated list of existing versions to include (only used with --versions-list list)")
	fs.StringVar(&latestVersionSha, "latest-version-sha", "", "git SHA for the target version")
	fs.StringVar(&versionsList, "versions-list", "local", "how to determine versions list: fetch (from remote), local (only local versions), list (use --existing-versions)")
	fs.Parse(args)

	if outputPath == "" {
		log.Fatal("output directory required (-o)")
	}

	if targetVersion == "" {
		log.Fatal("target version required (--target-version)")
	}

	if latestVersionSha == "" {
		log.Fatal("version SHA required (--latest-version-sha)")
	}

	// Validate versions-list mode
	if versionsList != "fetch" && versionsList != "local" && versionsList != "list" {
		log.Fatal("--versions-list must be one of: fetch, local, list")
	}

	var allVersions []string

	switch versionsList {
	case "fetch":
		// Fetch versions from GitHub API
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			log.Fatal("GITHUB_TOKEN environment variable required when using --versions-list fetch")
		}

		fetchedVersions, err := fetchVersionsFromGitHubAPI("grafana", "deployment_tools", token)
		if err != nil {
			log.Fatalf("Failed to fetch versions from GitHub API: %v", err)
		}

		// Target version might not be in remote yet, so ensure it's included
		allVersions = append([]string{targetVersion}, fetchedVersions...)
		// Remove duplicates
		seen := make(map[string]bool)
		var deduped []string
		for _, v := range allVersions {
			if !seen[v] {
				seen[v] = true
				deduped = append(deduped, v)
			}
		}
		allVersions = deduped

		fmt.Printf("Fetched %d versions from remote, including target: %v\n", len(allVersions), allVersions)

	case "local":
		// Scan for locally available versions
		var localVersions []string
		if _, err := os.Stat(outputPath); err == nil {
			// Check what version directories exist locally
			entries, err := os.ReadDir(outputPath)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() && isVersionDirectory(entry.Name()) {
						localVersions = append(localVersions, entry.Name())
					}
				}
			}
		}

		// Always include target version
		allVersions = append([]string{targetVersion}, localVersions...)
		// Remove duplicates
		seen := make(map[string]bool)
		var deduped []string
		for _, v := range allVersions {
			if !seen[v] {
				seen[v] = true
				deduped = append(deduped, v)
			}
		}
		allVersions = deduped

		fmt.Printf("Using locally available versions: %v\n", allVersions)

	case "list":
		// Use explicit list from --existing-versions
		if existingVersions == "" {
			allVersions = []string{targetVersion}
		} else {
			existingList := strings.Split(existingVersions, ",")
			// Trim whitespace from each version
			for i := range existingList {
				existingList[i] = strings.TrimSpace(existingList[i])
			}
			allVersions = append([]string{targetVersion}, existingList...)
		}

		fmt.Printf("Using explicit version list: %v\n", allVersions)
	}

	fmt.Printf("Generating versions.libsonnet with versions: %v (SHA: %s)\n", allVersions, latestVersionSha)

	// Generate template data
	data := VersionsTemplateData{
		Versions:      allVersions,
		LatestVersion: latestVersionSha,
	}

	// Load template from file
	versionsTemplate, err := loadTemplate("versions.libsonnet.tmpl")
	if err != nil {
		log.Fatalf("Failed to load versions template: %v", err)
	}

	// Parse template with helper functions
	versionsTmpl, err := template.New("versions").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(versionsTemplate)
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

	versionsTestTmpl, err := template.New("versions_test").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(versionsTestTemplate)
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
		"test-suite":               true, // old form of suite-path
		"test-suite-base":          true, // old form of suite-base
		"test-suite-name":          true, // old form of suite-name
		"test-suite-repo":          true, // old form of suite-repo-url
		"test-suite-repo-dirs":     true, // old form of suite-repo-dirs
		"test-suite-repo-token":    true, // old form of suite-repo-token
		"test-suite-revision":      true, // old form of suite-revision
		"test-env-vars":            true, // old form of test-env
		"pw-execute-cmd":           true, // old form of pw-execute
		"pw-prepare-cmd":           true, // old form of pw-prepare
		"codeowners-mapping":       true, // old form of slack-codeowners-mapping
		"dashboard":                true, // old form of run-dashboard
		"format":                   true, // old form of report-output
		"notify-passing":           true, // old form of slack-passing
		"run-trigger":              true, // old form of run-stage
		"test-trigger":             true, // old form of run-stage
		"trigger":                  true, // old form of run-stage
		"verbose":                  true, // old form of test-verbose
		"k6-cloud-project-id":      true, // old form of k6-cloud-project
		"report-format":            true, // old form of report-output
		"test-report-format":       true, // old form of report-output
		"suite-run-metrics":        true, // old form of run-metrics
		"suite-run-metrics-prefix": true, // old form of run-metrics-prefix
	}

	// Skip go-test related flags since they're not relevant for libsonnet
	goTestFlags := map[string]bool{
		"go-args":          true, // arguments to go test command - not needed for libsonnet
		"go-retries":       true, // number of retries for failed go tests - not needed for libsonnet
		"go-retry-delay":   true, // delay between go test retries - not needed for libsonnet
		"go-test-args":     true, // arguments to go test using arg flag - not needed for libsonnet
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

func generateScriptFlags(flags []FlagInfo, version string) string {
	var baseArrayParts []string
	var concatenationParts []string

	// Check if this is v0.6.11 (uses old deprecated flags)
	isLegacyVersion := version == "v0.6.11"

	// Always include basic required flags in the base array
	baseArrayParts = append(baseArrayParts, "                     'grafana-bench',")
	baseArrayParts = append(baseArrayParts, "                     'test',")

	// Add required string flags to base array (version-specific)
	if isLegacyVersion {
		// v0.6.11 uses old flags
		baseArrayParts = append(baseArrayParts, "                     '--grafana-url',")
		baseArrayParts = append(baseArrayParts, "                     grafanaURL,")
	} else {
		// v1.0.0+ uses new flags
		baseArrayParts = append(baseArrayParts, "                     '--service-url',")
		baseArrayParts = append(baseArrayParts, "                     serviceURL,")
	}
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
	if isLegacyVersion {
		// v0.6.11 uses old flag name
		baseArrayParts = append(baseArrayParts, "                     '--test-report-format',")
	} else {
		// v1.0.0+ uses new flag name
		baseArrayParts = append(baseArrayParts, "                     '--report-output',")
	}
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
		if isRequiredStringFlag(flag.Name) || isBaseArrayFlag(flag.Name) {
			return ""
		}
		// Handle special string escaping cases
		if strings.Contains(flag.Name, "pw-") {
			return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', std.escapeStringBash(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
		}
		if flag.Name == "run-dashboard" {
			return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', std.escapeStringBash(bench_options.%s)] else [])", camelCase, flag.Name, camelCase)
		}
		return fmt.Sprintf("+ (if bench_options.%s != '' then ['--%s', bench_options.%s] else [])", camelCase, flag.Name, camelCase)
	case "stringArray", "stringSlice":
		// Handle special cases
		if flag.Name == "suite-repo-dirs" {
			return fmt.Sprintf("+ std.flattenArrays([['--%s', dir] for dir in bench_options.%s])", flag.Name, camelCase)
		}
		return fmt.Sprintf("+ std.flattenArrays([['--%s', item] for item in bench_options.%s])", flag.Name, camelCase)
	case "stringToString":
		// Special handling for test-env: testEnv in libsonnet is an array of "key=value" strings for simplicity
		// Users write: testEnv: ['CI=1', 'VAR=$ENV_VAR'] and we pass them directly to --test-env
		if flag.Name == "test-env" {
			return "+ (if bench_options.testEnv != [] then ['--test-env', std.join(',', bench_options.testEnv)] else [])"
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
	// Production overrides: flags that should differ from CLI defaults in libsonnet
	productionOverrides := map[string]string{
		"report-output":          "'log'", // Production needs structured logs (CLI default is 'text')
		"grafana-admin-user":     "''",    // Production uses GRAFANA_ADMIN_USER env var (CLI default is 'admin')
		"grafana-admin-password": "''",    // Production uses GRAFANA_ADMIN_PASSWORD env var (CLI default is 'admin')
	}

	if override, exists := productionOverrides[flag.Name]; exists {
		return override
	}

	// For flags with CLI defaults, return empty/zero so they don't add unnecessary CLI flags
	// This keeps them discoverable in Suite but won't generate CLI flags when not overridden
	hasCliDefault := false
	switch flag.Type {
	case "bool":
		hasCliDefault = flag.DefaultValue == "true" // false is the zero value, so only true counts as a default
	case "string":
		hasCliDefault = flag.DefaultValue != "" && !isRequiredStringFlag(flag.Name)
	case "duration":
		hasCliDefault = flag.DefaultValue != "" && flag.DefaultValue != "0s"
	case "int":
		hasCliDefault = flag.DefaultValue != "" && flag.DefaultValue != "0"
	}

	if hasCliDefault {
		switch flag.Type {
		case "bool":
			return "false" // Return false so it won't add the flag
		case "duration":
			return "'0s'" // Return 0s so it won't add the flag
		case "int":
			return "0" // Return 0 so it won't add the flag
		default:
			return "''" // Return empty string so it won't add the flag
		}
	}

	// Standard handling for flags without CLI defaults
	switch flag.Type {
	case "bool":
		return "false"
	case "string":
		if isRequiredStringFlag(flag.Name) {
			return fmt.Sprintf("error 'must define %s'", strings.ReplaceAll(flag.Name, "-", " "))
		}
		return "''"
	case "stringArray", "stringSlice":
		return "[]"
	case "stringToString":
		// Special case for test-env which becomes testEnvVars in the libsonnet
		if flag.Name == "test-env" {
			return "[]"
		}
		return "{}"
	case "int":
		return "0"
	case "duration":
		return "'0s'"
	default:
		return "''"
	}
}

func isRequiredStringFlag(flagName string) bool {
	// These flags are required in the original libsonnet (have error defaults)
	requiredFlags := map[string]bool{
		"service-url": true, // Renamed from grafana-url in v1.0.0
		"suite-path":  true,
	}
	return requiredFlags[flagName]
}

// isBaseArrayFlag returns true for flags that are hardcoded in the base script
// array and must not also be emitted as optional concatenations.
func isBaseArrayFlag(flagName string) bool {
	return flagName == "report-output"
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

func fetchVersionsMain() {
	var repoOwner string
	var repoName string
	var token string

	// Parse args starting from index 2 since first arg is "fetchVersions"
	args := os.Args[2:]

	// Create a new FlagSet for parsing
	fs := flag.NewFlagSet("fetchVersions", flag.ExitOnError)
	fs.StringVar(&repoOwner, "deployment-tools-owner", "grafana", "GitHub repository owner")
	fs.StringVar(&repoName, "deployment-tools-repo", "deployment_tools", "GitHub repository name")
	fs.StringVar(&token, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub authentication token (defaults to GITHUB_TOKEN env var)")
	fs.Parse(args)

	if token == "" {
		log.Fatal("GitHub token required (--github-token or GITHUB_TOKEN env var)")
	}

	// Fetch directory contents using GitHub API
	versions, err := fetchVersionsFromGitHubAPI(repoOwner, repoName, token)
	if err != nil {
		log.Fatalf("Failed to fetch versions from GitHub API: %v", err)
	}

	// Output comma-separated list to stdout
	fmt.Print(strings.Join(versions, ","))
}

func fetchVersionsFromGitHubAPI(owner, repo, token string) ([]string, error) {
	// GitHub API URL for directory contents
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/ksonnet/lib/bench", owner, repo)
	
	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Add authentication header
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse JSON response
	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}
	
	// Filter for directories and extract version names
	var versions []string
	for _, item := range contents {
		if item.Type == "dir" && isVersionDirectory(item.Name) {
			versions = append(versions, item.Name)
		}
	}
	
	// Sort versions
	sort.Strings(versions)
	
	return versions, nil
}

func isVersionDirectory(name string) bool {
	// Regex pattern for semantic version folders (e.g., v1.2.3, v0.6.10)
	versionPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+.*$`)
	
	// Include: experimental, legacy, v*.*.* pattern
	return name == "experimental" || name == "legacy" || versionPattern.MatchString(name)
}

// scanVersionDirectories scans a bench directory and returns available version folders
func scanVersionDirectories(benchDir string) ([]string, error) {
	var versions []string

	entries, err := os.ReadDir(benchDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read bench directory %s: %w", benchDir, err)
	}

	// Regex pattern for semantic version folders (e.g., v1.2.3, v0.6.10)
	versionPattern := regexp.MustCompile(`^v\d+\.\d+\.\d+.*$`)

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Include: experimental, legacy, v*.*.* pattern
			if name == "experimental" || name == "legacy" || versionPattern.MatchString(name) {
				versions = append(versions, name)
			}
		}
	}

	// Sort versions consistently
	sort.Strings(versions)
	return versions, nil
}
