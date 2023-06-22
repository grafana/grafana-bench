package provisioner

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
)

// Sets up directory with configs needed for testing a grafana
// build. This method expects the BuildArtifactPath to exist on disk.
func setupGrafanaWorkdir(ctx context.Context, bc *buildcache.BuildCache, ps *ProvisionState) (string, error) {
	// verify build artifact exists in the buildcache
	resolved, err := bc.Resolve(ctx, buildcache.TypeBuild, ps.Build.ArtifactBuildName)
	if err != nil {
		return "", fmt.Errorf("build-cache: error checking for grafana executable : %w", err)
	}
	if !resolved {
		return "", fmt.Errorf("build-cache: grafana executable not found: %s", ps.Build.ArtifactBuildName)
	}

	// verify defaults.ini exists in the buildcache
	resolved, err = bc.Resolve(ctx, buildcache.TypeINI, ps.Build.ArtifactININame)
	if err != nil {
		return "", fmt.Errorf("build-cache: error checking for defaults.ini: %w", err)
	}
	if !resolved {
		return "", fmt.Errorf("build-cache: defaults.ini not found: %s", ps.Build.ArtifactININame)
	}

	// delete old workdir if exists
	if err := utils.Rm(ps.WorkDir); err != nil {
		return "", fmt.Errorf("provisioner: error deleting workdir: %w", err)
	}

	// copy template directory
	if err := utils.Cp(ps.TemplateDir, ps.WorkDir); err != nil {
		return "", fmt.Errorf("provisioner: error copying template directory: %s - %w", ps.TemplateDir, err)
	}

	// Copy executable into work dir
	executableDestination := path.Join(ps.WorkDir, ps.Build.ArtifactBuildName)
	if err := bc.Retrieve(ctx, buildcache.TypeBuild, ps.Build.ArtifactBuildName, executableDestination); err != nil {
		return "", err
	}

	// copy defaults.ini into work dir
	iniDestination := path.Join(ps.WorkDir, "conf", "defaults.ini")
	if err := bc.Retrieve(ctx, buildcache.TypeINI, ps.Build.ArtifactININame, iniDestination); err != nil {
		return "", err
	}

	// copy custom.ini into work dir
	if ps.CustomGrafanaINIPath != "" {
		customIniWorkPath := path.Join(ps.WorkDir, "conf", "custom.ini")
		if err := utils.Cp(ps.CustomGrafanaINIPath, customIniWorkPath); err != nil {
			return "", err
		}
	}

	return executableDestination, nil
}
