package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/env"
	"github.com/grafana/grafana-bench/bench/utils/version"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log = log.With("svc", "test-runner")

	runner, err := testRunnerFromArgs(log, os.Args[1:])
	if err != nil {
		log.Error("parsing parameters", "error", err)
		os.Exit(1)
	}

	log.Info("Bench run params",
		"testType", runner.Type.Name(),
		"tests", runner.Tests,
		"grafanaInstance", runner.GrafanaInstance.Host,
		"k6ProjectId", runner.K6CloudProjectID,
	)

	// ENTRYPOINT to start actually running the tests
	err = runner.Exec(context.Background())
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// read test runner specification from CLI args
func testRunnerFromArgs(log *slog.Logger, args []string) (*TestRunner, error) {
	var (
		testTrigger      string
		testType         string
		address          string
		username         string
		password         string
		machineSpec      string
		revision         string
		revisionFile     string
		tests            string
		k6CloudToken     string
		k6CloudProjectId string
		grafanaTimeout   time.Duration
		benchRevision    string
		dashboardURL     string
		verbose          bool
		cloudOutput      bool
	)

	fs := flag.NewFlagSet("test runner", flag.ContinueOnError)
	fs.StringVar(&testTrigger, "trigger", "local", "test trigger")
	fs.StringVar(&testType, "type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&address, "instance", "http://localhost:3000", "url to grafana instance")
	fs.DurationVar(&grafanaTimeout, "timeout", 30*time.Second, "timeout for waiting grafana to be live")
	fs.StringVar(&username, "user", "admin", "grafana user name. Can be overridden by the GRAFANA_USER environment variable")
	fs.StringVar(&password, "password", "admin", "grafana password. Can be overridden by the GRAFANA_PASSWORD environment variable")
	fs.StringVar(&machineSpec, "spec", "", "grafana instance machine spec")
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(&revision, "revision", "", "test suite revision. Has precedence over revision-file")
	fs.StringVar(&revisionFile, "revision-file", "", "path to a file with the test revision")
	fs.StringVar(&benchRevision, "bench-revision", "", "grafana bench revision")
	fs.StringVar(&k6CloudToken, "k6-cloud-token", "", "K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used")
	fs.StringVar(&k6CloudProjectId, "k6-cloud-project", "", "K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used")
	fs.BoolVar(&verbose, "verbose", true, "show k6 test outputs")
	fs.BoolVar(&cloudOutput, "cloud-output", false, "send output to GCK6. Requires setting the GCK6 project ID and access token.")
	fs.StringVar(&dashboardURL, "dashboard", "", "Template for the smoke test suite execution dashboard URL."+
		"\nSupports the substitution of the following variables:"+
		"\n    SuiteRun: identifier of the suite run"+
		"\nExample: http://localhost/dashboards?run={{.SuiteRun}}",
	)

	err := fs.Parse(args)
	if err != nil {
		return nil, err
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

	grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(address, username, password)
	if err != nil {
		return nil, err
	}

	// override grafana user and password from environment variables if they are set
	username = env.EnvOrDefault("GRAFANA_USER", username)
	password = env.EnvOrDefault("GRAFANA_PASSWORD", password)

	if k6CloudToken == "" {
		k6CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", "")
	}

	if k6CloudProjectId == "" {
		k6CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")
	}

	// the test is specified as an argument after the flags
	switch fs.NArg() {
	case 0:
		return nil, fmt.Errorf("tests must be specified")
	case 1:
		tests = fs.Arg(0)
	default:
		return nil, fmt.Errorf("expected one test argument got %d: %s", fs.NArg(), fs.Args())
	}

	testFiles, err := getTestFiles(tests)
	if err != nil {
		return nil, fmt.Errorf("getting test list: %w", err)
	}

	return NewTestRunner(
		log,
		verbose,
		cloudOutput,
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
	), nil
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
