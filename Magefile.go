//go:build mage
// +build mage

package main

import (
	"context"
	"path"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/utils"
)

// This file is a thin wrapper to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

// Function to set defaults for CLI.
func CLIServiceDefaults(ctx context.Context) *bench.BenchService {
	projectRoot := utils.GetWorkdir()
	artifactsPath := path.Join(projectRoot, "artifacts")
	GCSCredPath := path.Join(projectRoot, "GCP-infra-manager-828bbfa6f427.json")

	svc, err := bench.NewBenchService(ctx, projectRoot, artifactsPath, GCSCredPath, "bench-builds")
	if err != nil {
		panic(err)
	}
	return svc
}

var BenchService *bench.BenchService = CLIServiceDefaults(context.Background())

func TestME(ctx context.Context) error {
	// create a build with some defaults
	// do the build if it's not resolved
	build, err := builder.NewGrafanaBuild("branch:main", "darwin/arm64")
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := BenchService.Builder.Build(build)
		if err != nil {
			return err
		}
	}

	// provision the build

	// test the build

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
	b := bench.NewBenchRun(ctx, CLIServiceDefaults(ctx))
	return b.Run(ctx)
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
