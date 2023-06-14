package builder

import (
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
func (bs *BuilderService) New(grafanaRevision, arch string) (*Build, error) {
	gitRef, err := resolveGrafanaRevision(grafanaRevision)
	if err != nil {
		return nil, err
	}

	artifactName := getBuildArtifactName(gitRef, arch)

	// TODO check if build/ini exist in cache and autoresolve

	return &Build{
		BuilderService:  bs,
		Arch:            "linux/amd64",
		GrafanaRevision: gitRef,
		ArtifactName:    artifactName,
		Resolved:        false,
	}, nil
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
