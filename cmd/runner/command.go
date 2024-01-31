package runner

import (
	"context"
	"fmt"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/env"
	"github.com/grafana/grafana-bench/bench/utils/version"
	"github.com/grafana/grafana-bench/cmd"

)

// usage for test command
const usage = `
The bench test subcommand is a wrapper for running k6 tests against a grafana instance.

usage for test runner:

    bench test [options] --test-suite <test suite>


Examples:

    bench test --test-suite /path/to/test/folder

    bench test --test-type load --test-suite /path/to/test.js
`

// TestRunnerCommand implements the Command interface
type TestRunCommand struct {
	log *slog.Logger
	runner TestRunner
}

// NewTestRunnerCommand creates e test runner command using CLI arguments 
func NewTestRunnerCommand(log *slog.Logger, args []string)  (cmd.Command, error) {
	log = log.With("svc", "test-runner")
	var (
		testTrigger      string
		testType         string
		grafanaUrl       string
		grafanaUsername  string
		grafanaPassword  string
		machineSpec      string
		revision         string
		revisionFile     string
		testSuite        string
		k6CloudToken     string
		k6CloudProjectId string
		grafanaTimeout   time.Duration
		benchRevision    string
		dashboardURL     string
		verbose          bool
		k6CloudOutput    bool
	)

	fs := flag.NewFlagSet("test runner", flag.ExitOnError)
	// this function will be called when the help flag is passed
	fs.Usage = func() {
		fmt.Print(usage)
		fmt.Print("\nArguments\n")
		fs.PrintDefaults()
	}

	fs.StringVar(&testTrigger, "test-trigger", "local", "test trigger")
	fs.StringVar(&testType, "test-type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&grafanaUrl, "grafana-url", "http://localhost:3000", "url to grafana instance")
	fs.DurationVar(&grafanaTimeout, "grafana-timeout", 30*time.Second, "timeout for waiting grafana to be live")
	fs.StringVar(&grafanaUsername, "grafana-username", "admin", "grafana user name. Can be overridden by the GRAFANA_USER environment variable")
	fs.StringVar(&grafanaPassword, "grafana-password", "admin", "grafana password. Can be overridden by the GRAFANA_PASSWORD environment variable")
	fs.StringVar(&machineSpec, "machine-spec", "", "grafana instance machine spec")
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(&revision, "test-suite-revision", "", "test suite revision. Has precedence over test-suite-revision-file")
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
	fs.StringVar(&testSuite, "test-suite", "", "path to the tests to be executed." +
		"\nA single .js file or a directory can be specified." +
		"\nIf a directory is specified, all .js files in the directory and its sub-directories will be executed as tests.")

	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	if testSuite ==  "" {
		return nil, fmt.Errorf("tests must be specified")
	}

	trt, err := ParseTestType(testType)
	if err != nil {
		return nil, err
	}

	// If revision-file and revision are specified, revision has precedence
	if revision == "" && revisionFile != "" {
		revision, err = getTestRevision(revisionFile)
		if err != nil {
			return nil, fmt.Errorf("getting version from file %s: %w", revisionFile, err)
		}
	}

	if benchRevision == "" {
		benchRevision = version.BenchVersion()
	}

	grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(grafanaUrl, grafanaUsername, grafanaPassword)
	if err != nil {
		return nil, err
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

	testFiles, err := getTestFiles(testSuite)
	if err != nil {
		return nil, fmt.Errorf("getting test list: %w", err)
	}

	runner := NewTestRunner(
		log,
		verbose,
		k6CloudOutput,
		testTrigger,
		trt,
		testFiles,
		revision,
		k6CloudProjectId,
		k6CloudToken,
		grafanaInstance,
		grafanaTimeout,
		machineSpec,
		benchRevision,
		dashboardURL,
	)

	return &TestRunCommand{
		log:     log,
		runner: *runner,
	}, nil 
}

// Exec runs the TestRunnerCommand
func (c *TestRunCommand)Exec(ctx context.Context) error {
	// TODO: review attributes reported in this log message
	c.log.Info(
		"test runner params",
		"testType", c.runner.Type.Name(),
		"tests", c.runner.Tests,
		"grafanaInstance", c.runner.GrafanaInstance.Host,
		"k6ProjectId", c.runner.K6CloudProjectID,
	)

	return c.runner.Exec(ctx)
}

// read test revision from test file
func getTestRevision(revisionFile string) (string, error) {
	bytes, err := os.ReadFile(revisionFile)
	if err != nil {
		return "", fmt.Errorf("getting test version version from %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

// getTestFiles builds a list of k6 tests to run
// If Tests has a js extension run that single file otherwise assume it's
// a folder and glob all of the .js files in it recursively
// e.g.
// tests=dashboard_read.js will run dashboard_read.js
// tests=dashboards will run all files in dashboards/**.*.js
//
// If TestSuite is blank, assume we want to run everything in dist/**.*.js
func getTestFiles(tests string) ([]string, error) {
	// single file if we have .js extension
	if strings.Contains(tests, ".js") {
		exists, _ := utils.PathExists(tests)
		if !exists {
			return nil, fmt.Errorf("test file %s was not found", tests)
		}
		return []string{tests}, nil
	}

	files, err := utils.GlobByExtension(tests, ".js")
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no test files found at %s", tests)
	}

	return files, nil
}
