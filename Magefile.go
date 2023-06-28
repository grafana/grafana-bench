//go:build mage
// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
)

// This file is a thin wrapper to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

var BenchService *bench.BenchService = CLIServiceDefaults(context.Background())

// Get GoEnv from system running mage
var goEnv = utils.GetCompilerEnvInfo()

// Function to set defaults for CLI.
func CLIServiceDefaults(ctx context.Context) *bench.BenchService {
	execRoot := utils.Getwd()

	workPath := path.Join(execRoot, "work")
	buildCachePath := path.Join(workPath, "buildcache")

	GCSCredPath := path.Join(execRoot, "creds", "GCP-infra-manager-828bbfa6f427.json")
	K6CloudTokenPath := path.Join(execRoot, "creds", "k6cloud_jefflevinslunch_grafana_net")

	svc, err := bench.NewBenchService(ctx, workPath, buildCachePath, GCSCredPath, K6CloudTokenPath, "bench-builds")
	if err != nil {
		panic(err)
	}
	return svc
}

// Gets the architecture of the machine running Bench
func defaultArch() string {
	sys_os := goEnv["GOOS"]
	sys_arch := goEnv["GOARCH"]
	return fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
}

// Get environment variable or use default value
func envOrDefault(environmentVarName, defaultValue string) string {
	v := os.Getenv(environmentVarName)
	if v == "" {
		return defaultValue
	}

	return v
}

func TestME(ctx context.Context) error {
	grafanaRevision := envOrDefault("GRAFANA_REVISION", "branch:main")
	grafanaArch := envOrDefault("GRAFANA_ARCH", defaultArch())

	// TODO allow set provision type via cmd line
	//provisionDriver := envOrDefault("PROVISION_DRIVER", "local")

	// create a build with some defaults do the build if it's not resolved
	build, err := BenchService.Builder.New(ctx, grafanaRevision, grafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	// verify build exists in the cache
	resolved, err := BenchService.BuildCache.Resolve(ctx, buildcache.TypeBuild, build.ArtifactBuildName)
	if err != nil {
		return err
	}

	if resolved {
		fmt.Println("mage: build in cache")
	}

	ps, err := BenchService.Provisioner.New(ctx, provisioner.GCP, build)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	// TODO maybe on cancellation we destroy?

	ps.WaitForReady(ctx)

	return nil
	// test the build
	//testRun, err := BenchService.Tester.New(ctx, "jalevin/test", "dashboards/dashboard_create.js", true)
	//if err != nil {
	//  return err
	//}

	////// run the tests
	//err = ps.RunTests(ctx, testRun)
	//if err != nil {
	//  return err
	//}

	//// remove the build artifacts
	//return ps.Destroy(ctx)
}

func ContinueTest(ctx context.Context) error {
	state := os.Getenv("STATE")
	if state == "" {
		return fmt.Errorf("invalid state: \"%s\"", state)
	}

	ps, err := BenchService.Provisioner.ReadStateFile(state)
	if err != nil {
		return err
	}

	ps.WaitForReady(ctx)

	//ps.RunTests()

	return nil
}

// Destroy looks up the state and tears down a provision state
func Destroy(ctx context.Context) error {
	state := os.Getenv("STATE")
	if state == "" {
		return fmt.Errorf("invalid state: \"%s\"", state)
	}
	ps, err := BenchService.Provisioner.ReadStateFile(state)
	if err != nil {
		return err
	}

	return ps.Destroy(ctx)
}

// Runs test suit on already running instance of grafana
func Test(ctx context.Context) error {
	state := os.Getenv("STATE")

	if state == "" {
		return fmt.Errorf("invalid state: \"%s\"", state)
	}

	ps, err := BenchService.Provisioner.ReadStateFile(state)
	if err != nil {
		return err
	}

	// TODO START HERE
	// 1. test ensuring k6 instance can communicate with grafana instance
	// 2. figure out why first test command is failing!
	// 3. test executing tests on k6 instance
	// 4. test executing a different bundle

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "jalevin/test", "dashboards/dashboard_create.js", true)
	if err != nil {
		return err
	}

	// run the tests
	if err := ps.RunTests(ctx, testRun); err != nil {
		fmt.Println("error running tests:", err)
		fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}
	return nil
}

// Build builds a grafana binary and stores it in the artifacts folder
// usage: GRAFANA_REVISION=branch:k8s-proof-of-concept mage buildcommit
func Build(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.Build(ctx)
}

func ListBuilds(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.ListBuilds(ctx)
}

// Bench handles building, running, and benchmarking a commit.
// Defaults to using latest commit on Main.
// You can set the revision yourself. Usage:
// `GRAFANA_REVISION=branch:k8s-proof-of-concept mage bench`
// `GRAFANA_REVISION=commit:c116545e0ba005e10e318da96688bdae01439bf5 mage bench`
//
// If you would like to specify a custom configuration, you can either set the
// GRAFANA_CONFIG variable or place a custom.ini in the bench directory on disk
// usage: `INI=custom.ini mage bench`
func Bench(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.Bench(ctx)
}

// Build and run grafana, but wait for input to shutdown
func Run(ctx context.Context) error {
	grafanaRevision := envOrDefault("GRAFANA_REVISION", "branch:main")
	grafanaArch := envOrDefault("GRAFANA_ARCH", defaultArch())

	// create a build with some defaults do the build if it's not resolved
	build, err := BenchService.Builder.New(ctx, grafanaRevision, grafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	// verify build exists in the cache
	resolved, err := BenchService.BuildCache.Resolve(ctx, buildcache.TypeBuild, build.ArtifactBuildName)
	if err != nil {
		return err
	}

	if resolved {
		fmt.Println("mage: build in cache")
	}

	ps, err := BenchService.Provisioner.New(ctx, provisioner.Local, build)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	ps.WaitForReady(ctx)

	// wait for signal to kill grafana
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("Shutting down grafana process")
	return nil
	//b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	//return b.Run(ctx)
}

// TODO detail environment variables to set
func Help() {
	// ARCH
	// GRAFANA_REVISION
	// GRAFANA_CONFIG
	// TEST_SUITE_REVISION
	// TEST_SUITE
}
