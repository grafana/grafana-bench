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

	ps.WaitForReady(ctx)

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "jalevin/test", "dashboards/dashboard_create.js", true)
	if err != nil {
		return err
	}

	// run the tests
	err = ps.RunTests(ctx, testRun)
	if err != nil {
		return err
	}

	// remove the build artifacts
	return ps.Destroy(ctx)
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

// Runs test suit on already running instance of grafana
func Test(ctx context.Context, testSuite string) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	b.TestSuite = testSuite
	return b.Test(ctx)
}

// UpdateDeps updates build and test repos
func UpdateDeps(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.UpdateDeps()
}

// Resolve branch to latest commit of branch
func ResolveGrafanaRevision(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.ResolveGrafanaRevision()
}

// Resolve architecture and artifact names
func ResolveArch(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.ResolveArch()
}

// ResolveINI determines if there is a custom.ini to test a version of grafana
// with
func ResolveGrafanaINI(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.ResolveGrafanaINI()
}

// ResolveConfig resolves GrafanaCommit, Architecture, and Custom.ini. Use this
func ResolveConfig(ctx context.Context) error {
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.ResolveConfig(ctx)
}

// TODO detail environment variables to set
func Help() {
	// ARCH
	// GRAFANA_REVISION
	// GRAFANA_CONFIG
	// TEST_SUITE_REVISION
	// TEST_SUITE
}
