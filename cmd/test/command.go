package test

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/env"
	"github.com/spf13/cobra"
)

const examples = `
    bench test --test-suite /path/to/test/folder
    bench test --test-type load --test-suite /path/to/test.js"
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	log = log.With("svc", "test-runner")
	var (
		testTrigger       string
		testType          string
		grafanaUrl        string
		grafanaUsername   string
		grafanaPassword   string
		machineSpec       string
		testSuiteName     string
		testSuiteRevision string
		revisionFile      string
		testSuite         string
		testSuiteBase     string
		k6CloudToken      string
		k6CloudProjectId  string
		grafanaTimeout    time.Duration
		benchRevision     string
		dashboardURL      string
		verbose           bool
		k6CloudOutput     bool
	)

	cmd := cobra.Command{
		// test-suite is a mandatory option. highlight in the help
		Use:     "test --test-suite /path/to/test/suite",
		Short:   "bench test runner",
		Long:    "test subcommand is a wrapper for running k6 tests against a grafana instance",
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			trt, err := ParseTestType(testType)
			if err != nil {
				return err
			}

			// If revision-file and test-suite-revision are specified, test-suite-revision has precedence
			if testSuiteRevision == "" && revisionFile != "" {
				testSuiteRevision, err = getTestSuiteRevision(revisionFile)
				if err != nil {
					return fmt.Errorf("getting version from file %s: %w", revisionFile, err)
				}
			}

			if benchRevision == "" {
				benchRevision = bench.Revision()
			}

			grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(grafanaUrl, grafanaUsername, grafanaPassword)
			if err != nil {
				return err
			}

			// override grafana user and password from environment variables if they are set
			grafanaUsername = env.EnvOrDefault("GRAFANA_USER", grafanaUsername)
			grafanaPassword = env.EnvOrDefault("GRAFANA_PASSWORD", grafanaPassword)

			if k6CloudToken == "" {
				k6CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", "")
			}

			if k6CloudProjectId == "" {
				k6CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")
			}

			testSuitePath, testFiles, err := getTestFiles(testSuiteBase, testSuite)
			if err != nil {
				return fmt.Errorf("getting test files: %w", err)
			}

			// if the name of the test suite was not given, use the last element of the test path as name
			if testSuiteName == "" {
				testSuiteName = strings.TrimSuffix(path.Base(testSuitePath), ".js")
			}

			runner := NewTestRunner(
				log,
				verbose,
				k6CloudOutput,
				testTrigger,
				trt,
				testFiles,
				testSuiteName,
				testSuiteRevision,
				k6CloudProjectId,
				k6CloudToken,
				grafanaInstance,
				grafanaTimeout,
				machineSpec,
				benchRevision,
				dashboardURL,
			)

			// TODO: review attributes reported in this log message
			log.Info(
				"test runner params",
				"testType", runner.Type.Name(),
				"tests", runner.Tests,
				"grafanaInstance", runner.GrafanaInstance.Host,
				"k6ProjectId", runner.K6CloudProjectID,
			)

			return runner.Exec(cmd.Context())
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&testTrigger, "test-trigger", "local", "test trigger")
	fs.StringVar(&testType, "test-type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&grafanaUrl, "grafana-url", "http://localhost:3000", "url to grafana instance")
	fs.DurationVar(&grafanaTimeout, "grafana-timeout", 30*time.Second, "timeout for waiting grafana to be live")
	fs.StringVar(&grafanaUsername, "grafana-username", "admin", "grafana user name. Can be overridden by the GRAFANA_USER environment variable")
	fs.StringVar(&grafanaPassword, "grafana-password", "admin", "grafana password. Can be overridden by the GRAFANA_PASSWORD environment variable")
	fs.StringVar(&machineSpec, "machine-spec", "", "grafana instance machine spec")
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(&testSuiteRevision, "test-suite-revision", "", "test suite revision. Has precedence over test-suite-revision-file")
	fs.StringVar(&revisionFile, "test-suite-revision-file", "", "path to a file with the test suite revision")
	fs.StringVar(&benchRevision, "bench-revision", "", "grafana bench revision")
	fs.StringVar(&k6CloudToken, "k6-cloud-token", "", "K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used")
	fs.StringVar(&k6CloudProjectId, "k6-cloud-project", "", "K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used")
	fs.BoolVar(&verbose, "verbose", true, "show test outputs")
	fs.BoolVar(&k6CloudOutput, "k6-cloud-output", false, "send output to GCK6. Requires setting the GCK6 project ID and access token.")
	fs.StringVar(&dashboardURL, "dashboard", "", "Template for the smoke test suite execution dashboard URL."+
		"\nSupports the substitution of the following variables:"+
		"\n    SuiteRun: identifier of the suite run"+
		"\nExample: http://localhost/dashboards?run={{.SuiteRun}}",
	)
	fs.StringVar(&testSuite, "test-suite", "", "path to the tests to be executed."+
		"\nA single .js file or a directory can be specified."+
		"\nIf a directory is specified, all .js files in the directory and its sub-directories will be executed as tests.")
	cmd.MarkFlagRequired("test-suite")
	fs.StringVar(&testSuiteBase, "test-suite-base", "", "base directory for searching test suites."+
		"\nIf specified, it is prefixed to the --test-suite.")
	fs.StringVar(&testSuiteName, "test-suite-name", "", "test suite name. If not specified, the last component of --test-suite will be used."+
		"\nFor example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.")

	return &cmd
}

// read test suite revision from file
func getTestSuiteRevision(revisionFile string) (string, error) {
	bytes, err := os.ReadFile(revisionFile)
	if err != nil {
		return "", fmt.Errorf("getting test suite revision  from %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

// getTestFiles returns the path to the test suite and the list of tests to execute.
// The path prefixed with the basedir, which can be empty.
// If tests points to a file with a js extension run that single file.
// If it it points to a directory all of the .js files in it are recursively searched.
// tests=dashboard_read.js will run dashboard_read.js.
// tests=dashboards will run all files in dashboards/**.*.js.
func getTestFiles(baseDir string, tests string) (string, []string, error) {
	testSuitePath, err := filepath.Abs(path.Join(baseDir, tests))
	if err != nil {
		return "", nil, fmt.Errorf("getting path to test suite %w", err)
	}

	exists, _ := utils.PathExists(testSuitePath)
	if !exists {
		return "", nil, fmt.Errorf("test suite %s was not found", testSuitePath)
	}

	fileInfo, err := os.Stat(testSuitePath)
	if err != nil {
		return "", nil, fmt.Errorf("opening test suite at %s: %w", testSuitePath, err)
	}

	// test suite points to a directory
	if fileInfo.IsDir() {
		files, err := utils.GlobByExtension(testSuitePath, ".js")
		if err != nil {
			return "", nil, err
		}

		if len(files) == 0 {
			return "", nil, fmt.Errorf("no test files found at %s", testSuitePath)
		}

		return testSuitePath, files, nil
	}

	// is a file, we expect a single .js file
	testSuiteFile := path.Base(testSuitePath)
	if !strings.HasSuffix(testSuiteFile, ".js") {
		return "", nil, fmt.Errorf("expected a .js file got %s", testSuiteFile)
	}

	return testSuitePath, []string{testSuitePath}, nil
}
