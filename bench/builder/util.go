package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils/git"
	"github.com/magefile/mage/sh"
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
		return "", fmt.Errorf("Invalid GrafanaRevision format. Use `commit:e74e7fa` or `branch:main`")
	}

	t, val := pieces[0], pieces[1]

	if t != "commit" && t != "branch" {
		return "", fmt.Errorf("Invalid GrafanaRevision format. Use `commit:e74e7fa` or `branch:main`")
	}

	if t == "commit" && len(val) < 7 {
		return "", fmt.Errorf("Invalid GrafanaRevision format: %s commit ref must be at least 7 characters", grafanaRevision)
	}

	var commit string
	var err error
	if t == "branch" {
		fmt.Println("grafana: branch", val, "specified. Resolving latest commit")
		commit, err = git.ResolveLatestBranchCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
	} else if t == "commit" {
		fmt.Println("grafana: commit", val, "specified. Resolving commit")
		commit, err = git.ResolveFullCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
	}

	return commit, nil
}

// Downloads defaults.ini for a given build of Grafana to specified directory
func downloadIni(ctx context.Context, grafanaRevision, destination string) error {
	// get the ini for that commit of grafana if it doesn't exist
	// takes 7 chars to full commit hash
	iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", grafanaRevision)
	if err := sh.RunV("curl", iniUrl, "-o", destination); err != nil {
		return err
	}

	return nil
}

// generates the name of the build artifact for caching
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126-darwin-arm64
func getBuildArtifactName(grafanaGitRef, arch string) string {
	// darwin/arm64 -> darwin-arm64
	arch = strings.Replace(arch, "/", "-", -1)
	return fmt.Sprintf("%s-%s-grafana-server", grafanaGitRef, arch)
}

func getIniArtifactName(grafanaGitRef string) string {
	return fmt.Sprintf("%s_defaults.ini", grafanaGitRef)
}
