package provisioner

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type LocalDriver struct {
	LocalDir   string
	buildCache *buildcache.BuildCache
}

// Provision - provisions Grafana + test runner
func (l *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) error {
	executable, err := l.setupWorkdir(ctx, ps)
	if err != nil {
		return err
	}

	// boot

	return nil
}

// Check - checks if Grafana + test runner are ready
func (l *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) (bool, error) {
	// check if grafana is running
	return false, nil
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (l *LocalDriver) Teardown(ctx context.Context, ps *ProvisionState) error {
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
func (l *LocalDriver) setupWorkdir(ctx context.Context, ps *ProvisionState) (string, error) {
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

	// TODO this should probably be resolved by the time we're doing setup as part
	// of the build process
	iniName := provisioner.iniName(ps.GrafanaRevision)
	iniWorkPath := path.Join(ps.WorkDir, "conf", "defaults.ini")

	exists, err := l.buildCache.Resolve(ctx, buildcache.IniObj, iniName)
	if err != nil {
		return "", fmt.Errorf("build-cache: error retrieving ini artifact err: %w", err)
	}
	// if doesn't exist, write to workdir and call cache on it.
	if !exists {
		err := provisioner.GetBuildINI(ctx, ps.GrafanaRevision, iniWorkPath)
		if err != nil {
			return err
		}

		err = l.buildCache.Store(ctx, buildcache.IniObj, iniWorkPath, iniName)
		if err != nil {
			fmt.Println("build-cache: error storing ini artifact: ", err)
		}
		// get from cache
	} else {
		err := l.buildCache.Retrieve(ctx, buildcache.IniObj, iniName, iniWorkPath)
		if err != nil {
			return "", err
		}
	}

	// copy custom.ini into work dir
	if ps.CustomGrafanaINIPath != "" {
		customIniWorkPath := path.Join(ps.WorkDir, "conf", "custom.ini")
		if err := sh.Run("cp", ps.CustomGrafanaINIPath, customIniWorkPath); err != nil {
			return "", err
		}
	}

	// copy artifact
	exists, err = l.buildCache.Resolve(ctx, buildcache.BuildObj, ps.GrafanaArtifactName)
	if err != nil {
		return "", fmt.Errorf("build-cache: error retrieving build artifact err: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("build-cache: build artifact does not exist")
	}

	workExecutable := path.Join(b.ProjectRoot, "work", b.BuildArtifactName)
	if err := sh.RunV("cp", b.BuildCache.DiskPath(b.BuildArtifactName), workExecutable); err != nil {
		return "", err
	}
	return workExecutable, nil
}
