package main

import (
	"os"

	"log/slog"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils/env"
)

var benchRevision = "local"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log = log.With("svc", "init-test-runner")

	// Setup bench service with defaults for CLI
	if len(os.Args) != 6 {
		log.Error("Missing parameters. need 6 args; testType grafanaAddress username password tests", "argCount", len(os.Args))

		// one of these will panic and exit
		log.Error("arg[0]", "exec", os.Args[0])
		log.Error("arg[1]", "testType", os.Args[1])
		log.Error("arg[2]", "address", os.Args[2])
		log.Error("arg[4]", "username", os.Args[3])
		log.Error("arg[4]", "password", os.Args[4])
		log.Error("arg[5]", "tests", os.Args[5])
	}

	var (
		testType = os.Args[1]
		address  = os.Args[2]
		username = os.Args[3]
		password = os.Args[4]
		tests    = os.Args[5]
	)

	trt, err := ParseTestType(testType)
	if err != nil {
		log.Error(err.Error())
	}

	grafanaInstance, err := provisioner.NewReadOnlyGrafanaVM(address, username, password)
	if err != nil {
		log.Error(err.Error())
	}

	runner := &TestRunner{
		K6CloudToken:     env.EnvOrDefault("K6_CLOUD_TOKEN", ""),
		K6CloudProjectID: env.EnvOrDefault("K6_CLOUD_PROJECT_ID", ""),
		GrafanaInstance:  grafanaInstance,
	}

	log.Info("Bench run params",
		"testType", trt.Name(),
		"grafanaAddress", address,
		"grafanaUser", username,
		"tests", tests,
		"k6ProjectId", runner.K6CloudProjectID,
	)

	// ENTRYPOINT to start actually running the tests
	err = runner.Exec(trt)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
