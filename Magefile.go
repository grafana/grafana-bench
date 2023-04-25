//go:build mage
// +build mage

package main

import (
	"github.com/grafana/grafana-bench/bench"
)

// This file is a thin wrapper using mage to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

// Initialize config with runtime defaults
var Bencher *bench.Config = bench.NewBencher()

// Build builds a grafana binary and stores it in the artifacts folder
// usage: COMMIT=k8s-proof-of-concept mage buildcommit
func Build() error {
	return Bencher.Build()
}

// BenchCommit load tests a commit.
// If you don't set the commit environment variable, it will
// default to main and resolve the git hash. You can also set commit to
// be a branch and it will grab the latest commit for that branch.
// usage:
//
// COMMIT=k8s-proof-of-concept mage bench
//
// COMMIT=c116545e0ba005e10e318da96688bdae01439bf5 mage bench
//
// By default we will look for a custom.ini in the project root, however, you
// can also specify this by environment variable and path.
//
// usage: INI=custom.ini mage bench
func Bench() error {
	return Bencher.Bench()
}

// Used to test/verify behaviors

// Resolve branch to latest commit of branch
func ResolveGrafanaCommit() error {
	return Bencher.ResolveGrafanaCommit()
}

// Resolve architecture and artifact names
func ResolveArch() error {
	return Bencher.ResolveArch()
}

// ResolveINI determines if there is a custom.ini to test a version of grafana
// with
func ResolveINI() error {
	return Bencher.ResolveINI()
}

// ResolveConfig resolves GrafanaCommit, Architecture, and Custom.ini. Use this
func ResolveConfig() error {
	return Bencher.ResolveConfig()
}
