package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
func NewBuildService(buildcache *buildcache.BuildCache, localdir string) *BuilderService {
	buildSuiteDir := filepath.Join(localdir, "buildsuite")

	return &BuilderService{
		LocalDir:      localdir,
		BuildCache:    buildcache,
		buildSuiteDir: buildSuiteDir,
	}
}

// Creates a new build ref used to build Grafana
func (bs *BuilderService) New(ctx context.Context, grafanaRevision, arch string) (*Build, error) {

	if !validateArch(arch) {
		return nil, fmt.Errorf("build-service: invalid architecture %s", arch)
	}

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
