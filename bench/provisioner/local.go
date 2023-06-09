package provisioner

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type LocalDriver struct {
	LocalDir   string
	buildCache *buildcache.BuildCache
}

// Provision - provisions Grafana + test runner
func (p *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) error {
	executable, err := setupWorkdir(ctx, ps)
	if err != nil {
		return err
	}

	// boot

	return nil
}

// Check - checks if Grafana + test runner are ready
func (p *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) (bool, error) {
	// check if grafana is running
	return false, nil
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (p *LocalDriver) Teardown(ctx context.Context, ps *ProvisionState) error {
	return nil
}

// START HERE
// reworking the buildcache to work with ini + builds. this will probably break
// the CLI as is but it will allow us to use the build cache when provisioning.
// 1. finish getting local provisioner plumbed
// 2. fix cli so that we can test it
// 3. start working on making the cli use the provisioner

// setupWorkdir sets up directory with configs needed for testing a grafana
// build. This method expects the BuildArtifactPath to exist on disk.
func setupWorkdir(ctx context.Context, ps *ProvisionState) (string, error) {
	// verify executable exists
	exists, _ := utils.PathExists(ps.GrafanaPath)
	if !exists {
		return "", fmt.Errorf("grafana executable does not exist at %s", ps.GrafanaPath)
	}

	// delete old workdir if exists
	if err := sh.RunV("rm", "-rf", path.Join(ps.WorkDir)); err != nil {
		return "", err
	}

	// copy template directory
	if err := sh.RunV("cp", "-r", ps.TemplateDir, ps.WorkDir); err != nil {
		return "", err
	}

	err := BuildCache(ctx, ps, ps.WorkDir, ps.GrafanaRevision)
	if err != nil {
	}

	// get default.ini for that commit
	iniArtifact := fmt.Sprintf("%s_defaults.ini", b.GrafanaRevision)
	iniArtifactPath := path.Join(b.ProjectRoot, "artifacts", iniArtifact)
	exists, _ := utils.PathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		// takes 7 chars to full commit hash
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaRevision)
		if err := sh.RunV("curl", iniUrl, "-o", iniArtifactPath); err != nil {
			return "", err
		}
	}

	// copy ini to workdir
	iniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "defaults.ini")
	if err := sh.RunV("cp", iniArtifactPath, iniWorkPath); err != nil {
		return "", err
	}

	// copy custom.ini into work dir
	if b.GrafanaINIPath != "" {
		customIniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "custom.ini")
		if err := sh.Run("cp", b.GrafanaINIPath, customIniWorkPath); err != nil {
			return "", err
		}
	}

	// copy artifact
	workExecutable := path.Join(b.ProjectRoot, "work", b.BuildArtifactName)
	if err := sh.RunV("cp", b.BuildCache.DiskPath(b.BuildArtifactName), workExecutable); err != nil {
		return "", err
	}
	return workExecutable, nil

}
