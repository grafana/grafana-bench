// Package version provide information about the build version
package revision

import (
	"runtime/debug"
)

// bench is the bench build version.
var bench = ""

// BenchRevision returns the version of the currently executed bench
//
// If the bench is installed from the repository, go populates the build version in the debug
// information.
//
// When running it using go run or building using go build, the version is set by go to "(devel)"
//
// In this cases, the revision can be set using a compile flag when building the executable:
//
//	-ldflags='-X github.com/grafana/grafana-bench/pkg/revision.bench=<revision>'
//
// See for more details on go pseudo-version: https://github.com/golang/go/issues/50603
func BenchRevision() string {
	if bench != "" {
		return bench
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
