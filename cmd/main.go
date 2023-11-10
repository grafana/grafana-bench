package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
)

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	benchSvc, benchCfg := bench.NewBenchServiceOrPanic(ctx, log)

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

	log.Info("Bench run params",
		"testType", testType,
		"grafanaAddress", address,
		"grafanaUser", username,
		"tests", tests,
		"k6ProjectId", benchCfg.K6CloudProjectID,
	)

	if err := hgtest(ctx, log, benchSvc, benchCfg, testType, address, username, password, tests); err != nil {
		panic(err)
	}
}

func hgtest(ctx context.Context, log *slog.Logger, benchSvc *bench.BenchService, benchCfg *bench.BenchServiceCfg, testType, address, username, password, tests string) error {
	log = log.With("svc", "hgtest")

	// TODO remove this block. should get for free from config/service init
	// k6 cloud credentials
	benchSvc.Tester.K6CloudProjectId = os.Getenv("K6_CLOUD_PROJECT_ID")
	benchSvc.Tester.K6CloudToken = os.Getenv("K6_CLOUD_TOKEN")

	// populate grafana vm
	grafanaInstance := provisioner.NewReadOnlyGrafanaVM(address, username, password)

	grafanaVersion, err := provisioner.GetGrafanaBuildVersion(grafanaInstance)
	if err != nil {
		log.Error("error getting grafana version", "err", err)
		return fmt.Errorf("Error getting grafana version. exiting.. err: %w", err)
	}

	// use this to pass in the build version for logging,
	// but don't try to use the build object or bad things will happen fo sho
	build := &builder.Build{
		GrafanaRevision: grafanaVersion,
	}

	suiteRevision := "" // using precompiled tests. ignore
	tr, err := benchSvc.Tester.New(ctx, suiteRevision, testType, tests)
	if err != nil {
		return err
	}

	tr.SuiteRevision, err = tr.GetShortTestRevisionFromCompiled()
	if err != nil {
		return err
	}

	// create a new provision state
	ps, err := benchSvc.Provisioner.New(ctx, provisioner.HG, build, false)
	if err != nil {
		return err
	}

	// set identifier for suite run
	ps.Identifier = getNewSuiteIdentifier(build, ps, tr)
	log.Info("suite identifier", "identifier", ps.Identifier)

	// set vm
	ps.GrafanaInstance = grafanaInstance

	ps.WaitForReady(ctx)

	// run the tests
	if err := ps.RunTests(ctx, tr); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	return nil
}

// GetNewSuitedentifier creates an identifier to be used for
// building dashboards in hosted grafana
//
// smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
// load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func getNewSuiteIdentifier(b *builder.Build, ps *provisioner.ProvisionState, tr *tester.TestRun) string {
	// {type}-{time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-api-tests-%s-graf-%s",
		tr.Type,
		time.Now().UTC().Format("15:04:05"),
		tr.SuiteRevision,
		b.GrafanaRevision,
	)
}
