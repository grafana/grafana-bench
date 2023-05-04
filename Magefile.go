//go:build mage
// +build mage

package main

import (
	"github.com/grafana/grafana-bench/bench"
)

// This file is a thin wrapper to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

// Initialize config with runtime defaults
var Bencher *bench.Config = bench.NewBencher()

// Build builds a grafana binary and stores it in the artifacts folder
// usage: GRAFANA_REVISION=branch:k8s-proof-of-concept mage buildcommit
func Build() error {
	return Bencher.Build()
}

// BenchCommit load tests a commit.
// If you don't set the commit environment variable, it will
// default to main and resolve the git hash. You can also set commit to
// be a branch and it will grab the latest commit for that branch.
// usage:
//
// GRAFANA_REVISION=branch:k8s-proof-of-concept mage bench
//
// GRAFANA_REVISION=commit:c116545e0ba005e10e318da96688bdae01439bf5 mage bench
//
// By default we will look for a custom.ini in the project root, however, you
// can also specify this by environment variable and path.
//
// usage: INI=custom.ini mage bench
func Bench() error {
	return Bencher.Bench()
}

// Build and run grafana, but wait for input to shutdown
func Run() error {
	return Bencher.Run()
}

// Runs test suit on already running instance of grafana
func Test(testSuite string) error {
	Bencher.TestSuite = testSuite
	return Bencher.Test()
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
func ResolveConfig() error {
	return Bencher.ResolveConfig()
}

// TODO detail environment variables to set
func Help() {
	// ARCH
	// GRAFANA_REVISION
	// GRAFANA_CONFIG
	// TEST_SUITE_REVISION
	// TEST_SUITE
}
