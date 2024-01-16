package main

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/env"
)

var benchRevision = "local"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log = log.With("svc", "test-runner")

	runner, err := testRunnerFromArgs(log, os.Args)
	if err != nil {
		log.Error("parsing parameters", "error", err, "args", os.Args[1:])
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
	// arg[0] is the command name, checking for the rest of CLI arguments
	if len(args[1:]) != 7 {
		return nil, fmt.Errorf("invalid number of arguments. Expected 7 got %d", len(args))
	}

	var (
		testType    = args[1]
		address     = args[2]
		username    = args[3]
		password    = args[4]
		machineSpec = args[5]
		versionPath = args[6]
		tests       = args[7]
	)

	trt, err := ParseTestType(testType)
	if err != nil {
		return nil, err
	}

	exists, _ := utils.PathExists(tests)
	if !exists {
		return nil, fmt.Errorf("test file %s was not found", tests)
	}

	grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(address, username, password)
	if err != nil {
		return nil, err
	}

	k6CloudToken := env.EnvOrDefault("K6_CLOUD_TOKEN", "")
	k6CloudProjectId := env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")

	return NewTestRunner(
		log,
		trt,
		tests,
		versionPath,
		k6CloudProjectId,
		k6CloudToken,
		grafanaInstance,
		machineSpec,
	), nil
}