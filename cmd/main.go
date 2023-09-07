package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
)

func main() {
	// setup
	ctx := context.Background()
	benchSvc, benchCfg := bench.NewBenchServiceOrPanic(ctx)

	// Setup bench service with defaults for CLI
	if len(os.Args) != 6 {
		log.Println("Missing parameters. need 6 args; address port username password tests")

		// this will panic
		log.Println("address:", os.Args[1])
		log.Println("port:", os.Args[2])
		log.Println("username:", os.Args[3])
		log.Println("password:", os.Args[4])
		log.Println("tests:", os.Args[5])
	}

	var (
		address  = os.Args[1]
		port     = os.Args[2]
		username = os.Args[3]
		password = os.Args[4]
		tests    = os.Args[5]
	)

	if err := hgtest(ctx, benchSvc, benchCfg, address, port, username, password, tests); err != nil {
		panic(err)
	}
}

func hgtest(ctx context.Context, benchSvc *bench.BenchService, benchCfg *bench.BenchServiceCfg, address, port, username, password, tests string) error {
	// populate grafana vm
	grafanaInstance := &provisioner.VMInstance{
		// address is coming in including https://
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
	// but don't try to use this or bad things will happen fo sho
	b := &builder.Build{
		GrafanaRevision: grafanaVersion,
	}

	// Hosted Grafana driver won't resolve. It will just make sure tests exist
	// where they're supposed to
	// tests = dashboards
	// tests = dashboards/dashboard_create.js
	tr, err := benchSvc.Tester.New(ctx, "jalevin/test", tests, benchCfg.SmokeTest, true)
	if err != nil {
		return err
	}

	// create a new state
	provisionDriver := provisioner.HG
	ps, err := benchSvc.Provisioner.New(ctx, provisionDriver, b, false)
	if err != nil {
		return err
	}

	// override identifier
	ps.Identifier = GetNewIdentifier(b, ps, tr)
	// set the instance
	ps.GrafanaInstance = grafanaInstance

	ps.WaitForReady(ctx)

	// set project id to https://jefflevinslunch.grafana.net/a/k6-app/projects/3653020
	benchSvc.Tester.K6CloudProjectId = os.Getenv("K6_CLOUD_PROJECT_ID")
	benchSvc.Tester.K6CloudToken = os.Getenv("K6_CLOUD_TOKEN")

	// run the tests
	if err := ps.RunTests(ctx, tr); err != nil {
		log.Println("error running tests:", err)
		log.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}

	return nil
}

func GetNewIdentifier(b *builder.Build, ps *provisioner.ProvisionState, tr *tester.TestRun) string {
	t := time.Now().UTC().Format("15:04:05")
	sha := tr.GetShortTestRevision()
	// {time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-api-tests-%s-graf-%s", t, sha, b.GrafanaRevision)
}
