package builder

import (
	"fmt"
	"strings"
)

// validate arch string linux/amd64
func validateArch(archstring string) bool {
	parts := strings.Split(archstring, "/")
	if len(parts) != 2 {
		return false
	}

	os := parts[0]
	if os != "linux" && os != "darwin" && os != "windows" {
		return false
	}

	arch := parts[1]
	if arch != "amd64" && arch != "arm64" {
		return false
	}

	return true
}

// generates the name of the build artifact for caching
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126-darwin-arm64
func getArtifactBuildName(grafanaGitRef, arch string) string {
	// darwin/arm64 -> darwin-arm64
	arch = strings.Replace(arch, "/", "-", -1)
	return fmt.Sprintf("%s-%s-grafana-server", grafanaGitRef, arch)
}

// generates the name of the ini artifact for caching
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126_defaults.ini
func getArtifactININame(grafanaGitRef string) string {
	return fmt.Sprintf("%s_defaults.ini", grafanaGitRef)
}
