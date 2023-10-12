package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder/git"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Service used for managing builds
type BuilderService struct {
	Log           *slog.Logger
	LocalDir      string
	BuildCache    *buildcache.BuildCache
	buildSuiteDir string
}

// Creates a new build service and resolves the build suite
func NewBuildService(log *slog.Logger, buildcache *buildcache.BuildCache, localdir string) *BuilderService {
	// TODO remove this. don't have a need for nested folders
	buildSuiteDir := filepath.Join(localdir)

	return &BuilderService{
		Log:           log.With("svc", "builder"),
		LocalDir:      localdir,
		BuildCache:    buildcache,
		buildSuiteDir: buildSuiteDir,
	}
}

// Creates a new build ref used to build Grafana
func (bs *BuilderService) New(ctx context.Context, grafanaRevision, arch string) (*Build, error) {

	if !validateArch(arch) {
		return nil, fmt.Errorf("invalid architecture %s", arch)
	}

	gitRef, err := bs.ResolveGrafanaRevision(grafanaRevision)
	if err != nil {
		return nil, err
	}

	artifactBuildName := getArtifactBuildName(gitRef, arch)
	buildResolved, err := bs.BuildCache.Resolve(ctx, buildcache.TypeBuild, artifactBuildName)
	if err != nil {
		return nil, fmt.Errorf("build-service: could not resolve build: %w", err)
	}

	artifactININame := getArtifactININame(gitRef)
	iniResolved, err := bs.BuildCache.Resolve(ctx, buildcache.TypeINI, artifactININame)
	if err != nil {
		return nil, fmt.Errorf("build-service: could not resolve build: %w", err)
	}

	resolved := iniResolved && buildResolved

	return &Build{
		BuilderService:    bs,
		Arch:              arch,
		GrafanaRevision:   gitRef,
		ArtifactBuildName: artifactBuildName,
		ArtifactININame:   artifactININame,
		Resolved:          resolved,
	}, nil
}

// Resolves build suite. Always updates to latest version
func (bs *BuilderService) ResolveBuildSuite() error {
	exists, _ := utils.PathExists(bs.buildSuiteDir)

	// If exist, update to latest
	if exists {
		err := utils.DoInDir(bs.LocalDir, bs.buildSuiteDir, func() error {
			if err := sh.RunV("git", "checkout", "main"); err != nil {
				return fmt.Errorf("Error checking out grafana build repo %s", err)
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
		return fmt.Errorf("could not clone build suite: %w", err)
	}

	// clone path to dir
	bs.Log.Info("cloning build suite", "dir", bs.buildSuiteDir)
	if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-build", bs.buildSuiteDir); err != nil {
		return fmt.Errorf("Error checking out grafana test repo %s", err)
	}

	return nil
}

// Resolves branch or commit to full length ref supported values:
// "" - defaults to defaults to "branch:main" and will get the latest commit on
// main.
// "branch:yourbranch" - will get the latest commit from yourbranch
// "commit:abcdefg" - will get the full length commit. Must include at least 7
// characters of the ref
func (bs *BuilderService) ResolveGrafanaRevision(grafanaRevision string) (string, error) {
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
		bs.Log.Info("resolving grafana revision", "type", "branch", "branch", val)
		commit, err = git.ResolveLatestBranchCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
		bs.Log.Info("resolved revision", "commit", commit)
	} else if t == "commit" {
		bs.Log.Info("resolving grafana commit", "type", "commit", "commit", val)
		commit, err = git.ResolveFullCommit("grafana/grafana", val)
		if err != nil {
			return "", err
		}
		bs.Log.Info("resolved revision", "commit", commit)
	}

	return commit, nil
}

func (bs *BuilderService) ListBuilds(ctx context.Context) error {
	builds, err := bs.BuildCache.List(ctx, buildcache.TypeBuild)

	if err != nil {
		return err
	}

	fmt.Println("Builds")
	for _, b := range builds {
		fmt.Printf("%s: %s\n", b.Location, b.Name)
	}

	return nil
}
