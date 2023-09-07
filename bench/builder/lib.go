package builder

import (
	"fmt"
	"log"
	"strings"

	"github.com/grafana/grafana-bench/bench/builder/git"
)

// Resolves branch or commit to full length ref supported values:
// "" - defaults to defaults to "branch:main" and will get the latest commit on
// main.
// "branch:yourbranch" - will get the latest commit from yourbranch
// "commit:abcdefg" - will get the full length commit. Must include at least 7
// characters of the ref
func resolveGrafanaRevision(grafanaRevision string) (string, error) {
	if grafanaRevision == "" {
		grafanaRevision = "branch:main"
	}

	pieces := strings.Split(grafanaRevision, ":")
	if len(pieces) != 2 {
		return "", fmt.Errorf("builder: Invalid GrafanaRevision format. Use `commit:e74e7fa` or `branch:main`")
	}

	t, val := pieces[0], pieces[1]

	if t != "commit" && t != "branch" {
		return "", fmt.Errorf("builder: Invalid GrafanaRevision format. Use `commit:e74e7fa` or `branch:main`")
	}

	if t == "commit" && len(val) < 7 {
		return "", fmt.Errorf("builder: Invalid GrafanaRevision format: %s commit ref must be at least 7 characters", grafanaRevision)
	}

	var commit string
	var err error
	if t == "branch" {
		log.Println("builder: branch", val, "specified. Resolving latest commit")
		commit, err = git.ResolveLatestBranchCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
		log.Println("builder: branch", val, "resolved to commit", commit)
	} else if t == "commit" {
		log.Println("builder: commit", val, "specified. Resolving commit")
		commit, err = git.ResolveFullCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
		log.Println("builder: commit", val, "resolved to commit", commit)
	}

	return commit, nil
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
