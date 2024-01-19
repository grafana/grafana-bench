// Package version provide information about the build version
package version

import (
	"runtime/debug"
)

// BenchBuild is the grafana bench build version.  
var BenchBuild = ""

// BenchVersion returns the version of the currently executed bench
//
// The version can be set in the BenchBuild variable using a compile flag when building the executable:
//  -ldflags='-X github.com/grafana/grafana-bench/utils/version/BenchBuild=<version>'
//
// If the bench is not build but installed from the repository, go populates the build version in the debug
// information.
//
// When running it using go run the version is set by go to "(devel)"
// See for more details on go pseudo-version: https://github.com/golang/go/issues/50603
func BenchVersion() string {
	if BenchBuild != "" {
		return BenchBuild
	}

	// try to find build version from golang tool chain
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" {
			return bi.Main.Version
		}
	}

	// can't identify version, assume local development version
	return "(devel)"
}