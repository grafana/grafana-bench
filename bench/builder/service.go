package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Service used for managing builds
type BuilderService struct {
	LocalDir      string
	BuildCache    *buildcache.BuildCache
	buildSuiteDir string
}

// Creates a new build service and resolves the build suite
func NewBuildService(localdir string, buildcache *buildcache.BuildCache) *BuilderService {
	buildSuiteDir := filepath.Join(localdir, "buildsuite")

	return &BuilderService{
		LocalDir:      localdir,
		BuildCache:    buildcache,
		buildSuiteDir: buildSuiteDir,
	}
}

// Creates a new build ref used to build Grafana
func (bs *BuilderService) New(ctx context.Context, grafanaRevision, arch string) (*Build, error) {

	// TODO validate arch

	gitRef, err := resolveGrafanaRevision(grafanaRevision)
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
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to link the build suite to the service.
func (bs *BuilderService) ResolveBuildSuite() error {
	exists, _ := utils.PathExists(bs.buildSuiteDir)

	// If exist, update to latest
	if exists {
		err := utils.DoInDir(bs.LocalDir, bs.buildSuiteDir, func() error {
			if err := sh.RunV("git", "checkout", "main"); err != nil {
				return fmt.Errorf("build-service: Error checking out grafana build repo %s", err)
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
