//go:build mage
// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/provisioner"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))
var BenchService, BenchCfg = bench.NewBenchServiceOrPanic(context.Background(), log)

// Build builds a grafana binary and stores it in the artifacts folder
// usage: GRAFANA_REVISION=branch:k8s-proof-of-concept mage buildcommit
func Build(ctx context.Context) error {
	build, err := BenchService.Builder.New(ctx, BenchCfg.GrafanaRevision, BenchCfg.GrafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run a build of grafana. If the build is not in the cache, it will
// authmatically do the build.
// If you use this command with the environment variable PROVISION=local this
// will block until you exit which will shut down the local grafana process.
func Run(ctx context.Context) error {
	ps, err := getProvisionState(ctx, BenchCfg)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	ps.WaitForReady(ctx)

	// Wait for signal to kill grafana if we're using the local driver
	if BenchCfg.ProvisionDriver == provisioner.Local {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Info("Shutting down grafana process", "svc", "mage")
	}

	return nil
}

// Bench handles building, running, and benchmarking a commit.
// Defaults to using latest commit on Main.
// You can set the revision yourself. Usage:
// `GRAFANA_REVISION=branch:k8s-proof-of-concept mage bench`
// `GRAFANA_REVISION=commit:c116545e0ba005e10e318da96688bdae01439bf5 mage bench`
func Bench(ctx context.Context, testType string, testSuite string) error {
	ps, err := getProvisionState(ctx, BenchCfg)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	ps.WaitForReady(ctx)

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "", testType, testSuite)
	if err != nil {
		return err
	}

	// run the tests
	err = ps.RunTests(ctx, testRun)
	if err != nil {
		log.Error("error running tests", "svc", "mage", "err", err)
		if ps.Type == provisioner.GCP {
			fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
		}
	}

	// Provide a hint for people running with cloud driver so they don't have to
	// wait for terraform every time.
	if err != nil && BenchCfg.DestroyInfra && BenchCfg.ProvisionDriver == provisioner.GCP {
		fmt.Println("It appears that you're destroying state even though you got an error. you can preserve with DESTROY=false. Resources will be cleaned up every 24 hours automatically.")
	}

	if BenchCfg.DestroyInfra {
		return ps.Destroy(ctx)
	}

	fmt.Println(fmt.Sprintf("Preserving state. STATE=\"%s\"", ps.Identifier))

	return nil
}

// Runs test suite on already running instance of grafana. Requires state for
// operation
func Test(ctx context.Context, testType string, testSuite string) error {
	var ps *provisioner.ProvisionState
	var err error

	if BenchCfg.ProvisionState != "" {
		ps, err = BenchService.Provisioner.ReadStateFile(BenchCfg.ProvisionState)
		if err != nil {
			return err
		}
	} else {
		// if no state provided, assume local driver, port 3000
		ps = BenchService.Provisioner.NewLocalDevState(ctx, BenchCfg.GrafanaAddress, BenchCfg.GrafanaUser, BenchCfg.GrafanaPassword)
	}

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "", testType, testSuite)
	if err != nil {
		return err
	}

	// run the tests
	if err := ps.RunTests(ctx, testRun); err != nil {
		log.Error("error running tests", "svc", "mage", "err", err)
		if ps.Type == provisioner.GCP {
			fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
		}
	}
	return nil
}

// Destroy looks up the state and tears down a provision state
func Destroy(ctx context.Context) error {
	if BenchCfg.ProvisionState == "" {
		log.Error("invalid state", "svc", "mage", "identifier", BenchCfg.ProvisionState)
		return fmt.Errorf("invalid state: \"%s\"", BenchCfg.ProvisionState)
	}
	ps, err := BenchService.Provisioner.ReadStateFile(BenchCfg.ProvisionState)
	if err != nil {
		return err
	}

	return ps.Destroy(ctx)
}

// Lists builds in cache
func ListBuilds(ctx context.Context) error {
	return BenchService.Builder.ListBuilds(ctx)
}

// getProvisionState checks the buildconfig for state. If provided, reads the
// statefile, otherwise resolves the build and creates a new state.
func getProvisionState(ctx context.Context, cfg *bench.BenchServiceCfg) (*provisioner.ProvisionState, error) {

	// Read state if provided
	if BenchCfg.ProvisionState != "" {
		return BenchService.Provisioner.ReadStateFile(BenchCfg.ProvisionState)
	}

	if BenchCfg.ProvisionDriver != provisioner.Local {
		log.Warn("Provision driver is not local, defaulting to linux/amd64")
		BenchCfg.GrafanaArch = "linux/amd64"
	}

	// create a build with some defaults do the build if it's not resolved
	build, err := BenchService.Builder.New(ctx, BenchCfg.GrafanaRevision, BenchCfg.GrafanaArch)
	if err != nil {
		return nil, err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return nil, err
		}
	}

	// verify build exists in the cache
	resolved, err := BenchService.BuildCache.Resolve(ctx, buildcache.TypeBuild, build.ArtifactBuildName)
	if err != nil {
		return nil, err
	}

	if resolved {
		log.Info("build in cache", "svc", "mage")
	}

	return BenchService.Provisioner.New(ctx, BenchCfg.ProvisionDriver, build.GrafanaRevision, build.Arch, true)
}
