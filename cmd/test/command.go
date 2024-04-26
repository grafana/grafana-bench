package test

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/env"

	"github.com/spf13/cobra"
)

const examples = `
    bench test --test-suite /path/to/test/folder
    bench test --test-type load --test-suite /path/to/test.js"
`

const longDescription = `
test subcommand is a wrapper for running a suite of k6 tests against a grafana
instance.

The tests to be executed are defined by the --test-suite option.

Tests are parameterized via environment variables following grafana-api-tests
conventions[1].

Supports two kinds of test executions defined by the --test-type option:
* smoke: execute tests and reports failures (default)
* load: execute tests and report execution stats to GCK6

For load tests, if the --k6-cloud-output flag is true, the test results will be
sent to Grafana Cloud k6. The GCK6 credentials[2] must be provided as
environment variables or as arguments (--k6-cloud-project and --k6-cloud-token)

[1] https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables
[2] https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	log = log.With("svc", "test-runner")
	var (
		testTrigger       string
		testType          string
		reportFormat      string
		grafanaUrl        string
		grafanaUsername   string
		grafanaPassword   string
		machineSpec       string
		testSuiteName     string
		testSuiteRevision string
		testSuite         string
		revisionFile      string
		testSuiteBase     string
		k6CloudToken      string
		k6CloudProjectId  string
		grafanaTimeout    time.Duration
		benchRevision     string
		dashboardURL      string
		verbose           bool
		k6CloudOutput     bool
		k6UseTypescript   bool
	)

	cmd := cobra.Command{
		// test-suite is a mandatory option. highlight in the help
		Use:     "test --test-suite /path/to/test/suite",
		Short:   "bench test runner",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			trt, err := ParseTestType(testType)
			if err != nil {
				return err
			}

			// take the environment variable first
			if testSuiteRevision == "" {
				testSuiteRevision = env.EnvOrDefault("TEST_SUITE_REVISION", "")
			}

			// If revision-file and test-suite-revision are specified, test-suite-revision has precedence
			if testSuiteRevision == "" && revisionFile != "" {
				testSuiteRevision, err = getTestSuiteRevision(revisionFile)
				if err != nil {
					return fmt.Errorf("getting version from file %s: %w", revisionFile, err)
				}
			}

			if benchRevision == "" {
				benchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
			}

			// override grafana parameters from environment variables if they are set
			grafanaUrl = env.EnvOrDefault("GRAFANA_URL", grafanaUrl)
			grafanaUsername = env.EnvOrDefault("GRAFANA_USER", grafanaUsername)
			grafanaPassword = env.EnvOrDefault("GRAFANA_PASSWORD", grafanaPassword)

			grafanaInstance, err := grafana.NewInstance(
				grafanaUrl,
				grafanaUsername,
				grafanaPassword,
				grafana.WithTimeout(grafanaTimeout),
			)
			if err != nil {
				return err
			}

			if k6CloudToken == "" {
				k6CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", "")
			}

			if k6CloudProjectId == "" {
				k6CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")
			}

			// if the name of the test suite was not given, use the last element of the test suit path as name
			if testSuiteName == "" {
				defaultTestSuiteName := strings.TrimSuffix(path.Base(testSuite), path.Ext(testSuite))
				testSuiteName = env.EnvOrDefault("TEST_SUITE_NAME", defaultTestSuiteName)
			}

			if testSuiteBase == "" {
				testSuiteBase, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("getting work directory %w", err)
				}
			}

			suite := executor.TestSuite{
				Name:     testSuiteName,
				BaseDir:  testSuiteBase,
				Path:     testSuite,
				Revision: testSuiteRevision,
			}

			executor := NewK6TestExecutor(
				log,
				verbose,
				k6UseTypescript,
				k6CloudOutput,
				k6CloudToken,
				k6CloudProjectId,
			)

			runner := NewTestRunner(
				log,
				testTrigger,
				grafanaInstance,
				machineSpec,
				benchRevision,
				dashboardURL,
				executor,
				reportFormat,
			)

			// TODO: review attributes reported in this log message
			log.Info(
				"test runner params",
				"testType", testType,
				"grafanaInstance", runner.GrafanaInstance.Url(),
				"k6ProjectId", k6CloudProjectId,
			)

			return runner.Exec(cmd.Context(), trt, suite)
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&testTrigger, "test-trigger", "local", "test trigger")
	fs.StringVar(&testType, "test-type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(
		&reportFormat,
		"test-report-format",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'." +
		"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
	fs.StringVar(
		&grafanaUrl,
		"grafana-url",
		"http://localhost:3000",
		"url to grafana instance. Overridden by the GRAFANA_URL environment variable",
	)
	fs.DurationVar(
		&grafanaTimeout,
		"grafana-timeout",
		grafana.DefaultGrafanaTimeout,
		"timeout for waiting grafana to be live",
	)
	fs.StringVar(
		&grafanaUsername,
		"grafana-username",
		"admin",
		"grafana user name. Overridden by the GRAFANA_USER environment variable",
	)
	fs.StringVar(
		&grafanaPassword,
		"grafana-password",
		"admin",
		"grafana password. Overridden by the GRAFANA_PASSWORD environment variable",
	)
	fs.StringVar(&machineSpec, "machine-spec", "", "grafana instance machine spec")
	fs.StringVar(
		&revisionFile,
		"test-suite-revision-file",
		"",
		"path to a file with the test suite revision",
	)
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(
		&testSuiteRevision,
		"test-suite-revision",
		"",
		"test suite revision. If not set TEST_SUITE_REVISION environment variable is used",
	)
	fs.StringVar(
		&benchRevision,
		"bench-revision",
		"",
		"grafana bench revision. If not set BENCH_REVISION environment variable is used.",
	)
	fs.StringVar(
		&k6CloudToken,
		"k6-cloud-token",
		"",
		"K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used",
	)
	fs.StringVar(
		&k6CloudProjectId,
		"k6-cloud-project",
		"",
		"K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used",
	)
	fs.BoolVar(
		&k6UseTypescript,
		"k6-use-typescript",
		false,
		"run k6 typescript tests. Typescript tests are compiled before execution.",
	)
	fs.BoolVar(&verbose, "verbose", true, "show test outputs")
	fs.BoolVar(
		&k6CloudOutput,
		"k6-cloud-output",
		false,
		"send output to GCK6. Requires setting the GCK6 project ID and access token.",
	)
	fs.StringVar(
		&dashboardURL,
		"dashboard",
		"",
		"Template for the smoke test suite execution dashboard URL."+
			"\nSupports the substitution of the following variables:"+
			"\n    SuiteRun: identifier of the suite run"+
			"\nExample: http://localhost/dashboards?run={{.SuiteRun}}",
	)
	fs.StringVar(&testSuite, "test-suite", "", "path to the tests to be executed."+
		"\nThe path must be relative to the base dir (which defaults to the current directory)."+
		"\nA single .js file or a directory can be specified."+
		"\nIf a directory is specified, all .js files in the directory and its sub-directories will be executed.")
	cmd.MarkFlagRequired("test-suite")
	fs.StringVar(
		&testSuiteBase,
		"test-suite-base",
		"",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --test-suite.",
	)
	fs.StringVar(
		&testSuiteName,
		"test-suite-name",
		"",
		"test suite name. If not specified, TEST_SUITE_NAME environment variable is used."+
			"\nDefaults to the last component of --test-suite."+
			"\nFor example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.",
	)

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
