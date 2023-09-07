package builder

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Build represents a build of Grafana.
// Passed to provisioner to deploy Grafana
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

	// Maybe deprecate
	ArtifactBuildName string `json:"artifactBuildName"`
	// Maybe deprecate
	ArtifactININame string `json:"ArtifactININame"`

	// Determines whether build is complete
	Resolved bool `json:"resolved"`
}

// Synchronous method to build grafana.
// checks if build is in the cache or skip to force build
func (b *Build) Run(ctx context.Context) error {
	// ensure build suite exists and up to date
	if err := b.ResolveBuildSuite(); err != nil {
		return err
	}

	// check if build in cache first

	// do the build
	err := utils.DoInDir(b.LocalDir, b.buildSuiteDir, func() error {
		// cmd - note, verbose and distro must be provided at the end of the command
		cmd := []string{"run", "./cmd", "backend", "build",
			fmt.Sprintf("--distro=%s", b.Arch),
			fmt.Sprintf("--grafana-ref=%s", b.GrafanaRevision),
			"--verbose",
		}

		fmt.Println("builder: running command go", strings.Join(cmd, " "))

		err := sh.RunV("go", cmd...)
		return err
	})
	if err != nil {
		return err
	}

	// cache the build
	grafanaExecutablePath := path.Join(b.buildSuiteDir, "bin", b.Arch, "grafana")
	if err := b.BuildCache.StoreFile(ctx, buildcache.TypeBuild, grafanaExecutablePath, b.ArtifactBuildName); err != nil {
		return err
	}

	// get the default.ini
	iniString, err := b.GetDefaultINI(ctx)
	if err != nil {
		return err
	}

	// cache the ini
	err = b.BuildCache.StoreBytes(ctx, buildcache.TypeINI, iniString, getArtifactININame(b.GrafanaRevision))
	if err != nil {
		return err
	}

	// set build to resolved
	b.Resolved = true

	return nil
}

// Gets a presigned url for the build
func (b *Build) GetPresignedUrl(ctx context.Context) (string, error) {
	return b.BuildCache.GetPresignedUrl(ctx, buildcache.TypeBuild, b.ArtifactBuildName)
}

// Gets the ini for that commit of grafana if it doesn't exist
// takes 7 chars to full commit hash
func (b *Build) GetDefaultINI(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaRevision)

	response, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
