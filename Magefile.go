//go:build mage
// +build mage

package main

import (
	"context"

	"github.com/grafana/grafana-bench/bench"
)

// This file is a thin wrapper to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

// Initialize config with runtime defaults
var Bencher *bench.Config = bench.NewBencher()

// Build builds a grafana binary and stores it in the artifacts folder
// usage: GRAFANA_REVISION=branch:k8s-proof-of-concept mage buildcommit
func Build(ctx context.Context) error {
	return Bencher.Build(ctx)
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
	return Bencher.Bench(ctx)
}

// Build and run grafana, but wait for input to shutdown
func Run(ctx context.Context) error {
	return Bencher.Run(ctx)
}

// Runs test suit on already running instance of grafana
func Test(ctx context.Context, testSuite string) error {
	Bencher.TestSuite = testSuite
	return Bencher.Test(ctx)
}

// UpdateDeps updates build and test repos
func UpdateDeps() error {
	return Bencher.UpdateDeps()
}

// Resolve branch to latest commit of branch
func ResolveGrafanaRevision() error {
	return Bencher.ResolveGrafanaRevision()
}

// Resolve architecture and artifact names
func ResolveArch() error {
	return Bencher.ResolveArch()
}

// ResolveINI determines if there is a custom.ini to test a version of grafana
// with
func ResolveGrafanaINI() error {
	return Bencher.ResolveGrafanaINI()
}

// ResolveConfig resolves GrafanaCommit, Architecture, and Custom.ini. Use this
func ResolveConfig(ctx context.Context) error {
	return Bencher.ResolveConfig(ctx)
}

// TODO detail environment variables to set
func Help() {
	// ARCH
	// GRAFANA_REVISION
	// GRAFANA_CONFIG
	// TEST_SUITE_REVISION
	// TEST_SUITE
}
