package builder

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

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

	ArtifactName string

	// Determines whether build is complete
	Resolved bool `json:"resolved"`
}

// Synchronous method to build grafana. Checks buildcache and returns nil if build is already
// resolved.
func (b *Build) Run(ctx context.Context) error {

	resolved, err := b.BuildCache.Resolve(ctx, buildcache.BuildObj, b.ArtifactName)
	if err != nil && resolved {
		b.Resolved = true
		return nil
	}

	// ensure build suite exists and up to date
	if err := b.ResolveBuildSuite(); err != nil {
		return err
	}

	// run command
	err = utils.DoInDir(b.LocalDir, b.buildSuiteDir, func() error {
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
	grafanaExecutablePath := path.Join(b.buildSuiteDir, "bin", b.Arch, "grafana")
	if err := b.BuildCache.StoreFile(ctx, buildcache.BuildObj, grafanaExecutablePath, b.ArtifactName); err != nil {
		return err
	}

	// get the default.ini
	iniString, err := b.GetDefaultINI(ctx)
	if err != nil {
		return err
	}

	// cache the ini
	err = b.BuildCache.StoreBytes(ctx, buildcache.IniObj, iniString, getIniArtifactName(b.GrafanaRevision))
	if err != nil {
		return err
	}

	// set build to resolved
	b.Resolved = true

	return nil
}

// Gets a presigned url for the build
func (b *Build) GetPresignedUrl(ctx context.Context) (string, error) {
	return b.BuildCache.GetPresignedUrl(ctx, buildcache.BuildObj, b.ArtifactName)
}

func (b *Build) GetDefaultINI(ctx context.Context) ([]byte, error) {
	// get the ini for that commit of grafana if it doesn't exist
	// takes 7 chars to full commit hash
	url := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaRevision)

	response, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
