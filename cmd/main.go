package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
)

func main() {
	// setup
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// START HERE
	// 1. plumb the logger through to services
	// 2. update log statements in this file
	// 3. use main logger in provisioner
	// 4. figure out why we're not exiting from docker run script
	// 5. plumb test type through to provisioner

	benchSvc, benchCfg := bench.NewBenchServiceOrPanic(ctx)

	// Setup bench service with defaults for CLI
	if len(os.Args) != 6 {
		log.Error("Missing parameters. need 6 args; address port username password tests")

		// this will panic
		log.Info("address:", os.Args[1])
		log.Info("port:", os.Args[2])
		log.Info("username:", os.Args[3])
		log.Info("password:", os.Args[4])
		log.Info("tests:", os.Args[5])
	}

	var (
		// TODO get this as arg
		testType = "smoke"
		address  = os.Args[1]
		port     = os.Args[2]
		username = os.Args[3]
		password = os.Args[4]
		tests    = os.Args[5]
	)

	if err := hgtest(ctx, benchSvc, benchCfg, testType, address, port, username, password, tests); err != nil {
		panic(err)
	}
}

func hgtest(ctx context.Context, benchSvc *bench.BenchService, benchCfg *bench.BenchServiceCfg, testType, address, port, username, password, tests string) error {
	// k6 cloud credentials
	benchSvc.Tester.K6CloudProjectId = os.Getenv("K6_CLOUD_PROJECT_ID")
	benchSvc.Tester.K6CloudToken = os.Getenv("K6_CLOUD_TOKEN")

	// populate grafana vm
	grafanaInstance := &provisioner.VMInstance{
		Address:         strings.TrimPrefix(address, "https://"),
		ServicePort:     port,
		GrafanaUser:     username,
		GrafanaPassword: password,
	}

	grafanaVersion, err := provisioner.GetGrafanaBuildVersion(grafanaInstance)
	if err != nil {
		log.Println("Error getting grafana version:", err)
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
	ps.Identifier = GetNewSuiteIdentifier(build, ps, tr)
	// set vm
	ps.GrafanaInstance = grafanaInstance

	ps.WaitForReady(ctx)

	// run the tests
	if err := ps.RunTests(ctx, tr); err != nil {
		log.Println("error running tests:", err)
	}

	return nil
}

// GetNewSuitedentifier creates an identifier to be used for
// building dashboards in hosted grafana
//
// smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
// load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func GetNewSuiteIdentifier(b *builder.Build, ps *provisioner.ProvisionState, tr *tester.TestRun) string {
	// {type}-{time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-api-tests-%s-graf-%s",
		tr.Type,
		time.Now().UTC().Format("15:04:05"),
		tr.SuiteRevision,
		b.GrafanaRevision,
	)
}
