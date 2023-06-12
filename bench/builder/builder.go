package builder

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/git"
	"github.com/magefile/mage/sh"
)

type BuilderService struct {
	LocalDir      string
	BuildCache    *buildcache.BuildCache
	buildSuiteDir string
}

func NewBuildService(localdir string, buildcache *buildcache.BuildCache) *BuilderService {
	buildSuiteDir := filepath.Join(localdir, "buildsuite")

	return &BuilderService{
		LocalDir:      localdir,
		BuildCache:    buildcache,
		buildSuiteDir: buildSuiteDir,
	}
}

// Synchronous method to build grafana
func (bs *BuilderService) Build(ctx context.Context, b *Build) error {
	// resolve buildsuite
	if err := bs.ResolveBuildSuite(); err != nil {
		return err
	}

	// run command
	err := utils.DoInDir(bs.LocalDir, bs.buildSuiteDir, func() error {
		// Note, verbose and distro must be provided at the end of the command
		err := sh.RunV("go", "run", "./cmd", "backend", "build",
			fmt.Sprintf("--distro=%s", b.Arch),
			fmt.Sprintf("--grafana-ref=%s", b.GrafanaRevision),
			"--verbose")
		return err
	})
	if err != nil {
		return err
	}

	// cache the build
	grafanaExecutablePath := path.Join(bs.buildSuiteDir, "bin", b.Arch, "grafana")
	if err := bs.BuildCache.Store(ctx, buildcache.BuildObj, grafanaExecutablePath, b.BuildArtifactName); err != nil {
		return err
	}

	// set build to resolved
	b.Resolved = true

	return nil
}

// Build represents a build of Grafana.
// Should be passed to provisioner to deploy Grafana
type Build struct {
	*BuilderService

	// UUID for the build
	Identifier string `json:"identifier"`

	// Destination to store work files for the build
	WorkDir string

	// Architecture for the build
	Arch string `json:"arch"`

	// Branch or commit of Grafana to run. prefix the type that you're going to
	// provide. e.g. "branch:k8s-proof-of-concept" or "commit:e74e7fa"
	// commit refs must be 7 characters or longer
	GrafanaRevision string `json:"grafanaRevision"`

	// Short name to reference
	GrafanaIniPath string `json:"grafanaIni"`

	BuildArtifactName string

	// Determines whether build is complete
	Resolved bool `json:"resolved"`
}

// Creates a new build ref used to build Grafana
func NewGrafanaBuild(grafanaRevision, arch string) (*Build, error) {
	gitRef, err := resolveGrafanaRevision(grafanaRevision)
	if err != nil {
		return nil, err
	}

	buildArtifactName := getBuildArtifactName(gitRef, arch)

	return &Build{
		Arch:              "linux-amd64",
		GrafanaRevision:   gitRef,
		BuildArtifactName: buildArtifactName,
		Resolved:          false,
	}, nil
}

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

// Resolves build suite. Always updates to latest version
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to linnk the build suite to the service.
func (bs *BuilderService) ResolveBuildSuite() error {
	exists, _ := utils.PathExists(bs.buildSuiteDir)

	// If exist, update to latest
	if exists {
		err := utils.DoInDir(bs.LocalDir, bs.buildSuiteDir, func() error {
			if err := sh.RunV("git", "checkout", "main"); err != nil {
				return fmt.Errorf("Error checking out grafana test repo %s", err)
			}

			if err := sh.RunV("git", "pull"); err != nil {
				return err
			}

			return nil
		})
		return err
	}

	// ensure path exists
	if err := os.MkdirAll(bs.buildSuiteDir, 0755); err != nil {
		return fmt.Errorf("build-service: could not clone build suite: %w", err)
	}

	// clone path to dir
	fmt.Println("build-service: cloning build suite")
	if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-build", bs.buildSuiteDir); err != nil {
		return fmt.Errorf("Error checking out grafana test repo %s", err)
	}

	return nil
}

func getBuildArtifactName(grafanaGitRef, arch string) string {
	return fmt.Sprintf("grafana-server-%s-%s", grafanaGitRef, strings.Replace(arch, "/", "-", -1))
}
