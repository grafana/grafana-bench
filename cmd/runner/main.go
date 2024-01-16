package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"log/slog"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/env"
)

var benchRevision = "local"

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
		testType     string
		address      string
		username     string
		password     string
		machineSpec  string
		revision     string
		revisionFile string
		tests        string
	)

	fs := flag.NewFlagSet("test runner", flag.ContinueOnError)
	fs.StringVar(&testType, "type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&address, "instance", "http://localhost:3000", "url to grafana instance")
	fs.StringVar(&username, "user", "admin", "grafana user name")
	fs.StringVar(&password, "password", "admin", "grafana password")
	fs.StringVar(&machineSpec, "spec", "", "grafana instance machine spec")
	fs.StringVar(&revision, "revision", "", "test revision. Has precedence over version-file")
	fs.StringVar(&revisionFile, "revision-file", "", "path to a file with the test revision")
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
			return nil, fmt.Errorf("getting version from file %s: w", revisionFile, err)
		}
	}

	grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(address, username, password)
	if err != nil {
		return nil, err
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
	exists, _ := utils.PathExists(tests)
	if !exists {
		return nil, fmt.Errorf("test file %s was not found", tests)
	}

	k6CloudToken := env.EnvOrDefault("K6_CLOUD_TOKEN", "")
	k6CloudProjectId := env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")

	return NewTestRunner(
		log,
		trt,
		tests,
		revision,
		k6CloudProjectId,
		k6CloudToken,
		grafanaInstance,
		machineSpec,
	), nil
}


// read test revision from test file
func getTestRevision(revisionFile string) (string, error) {
	bytes, err := os.ReadFile(revisionFile)
	if err != nil {
		return "", fmt.Errorf("getting test version version from %s: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}
