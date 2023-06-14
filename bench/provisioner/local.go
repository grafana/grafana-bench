package provisioner

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type LocalDriver struct {
	LocalDir   string
	buildCache *buildcache.BuildCache
}

func NewLocalDriver(localDir string, buildCache *buildcache.BuildCache) *LocalDriver {
	return &LocalDriver{
		LocalDir:   localDir,
		buildCache: buildCache,
	}
}

// Provision - provisions Grafana + test runner
func (l *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {
	executable, err := l.setupWorkdir(ctx, ps)
	if err != nil {
		return func() error { return nil }, err
	}

	cmd := exec.Command(executable, "server")

	// function to return so we can kill the process
	killFunc := func() error {
		err := cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("ERROR killing grafana PID: %w", err)
		}
		return nil
	}

	err = utils.DoInDir(ps.WorkDir, "work", func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})

	if err != nil {
		return killFunc, err
	}

	return killFunc, nil
}

// Check - checks if Grafana + test runner are ready
func (l *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) error {
	// check if grafana is running
	return nil
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (l *LocalDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
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

	// TODO Move all of the logic to get the INI into the build
	iniName := IniFilename(ps.GrafanaRevision)
	iniWorkPath := path.Join(ps.WorkDir, "conf", "defaults.ini")

	exists, err := l.buildCache.Resolve(ctx, buildcache.IniObj, iniName)
	if err != nil {
		return "", fmt.Errorf("build-cache: error retrieving ini artifact err: %w", err)
	}
	// if doesn't exist, write to workdir and call cache on it.
	if !exists {
		err := GetBuildINI(ctx, ps.GrafanaRevision, iniWorkPath)
		if err != nil {
			return "", err
		}

		err = l.buildCache.StoreFile(ctx, buildcache.IniObj, iniWorkPath, iniName)
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

	executableDestination := path.Join(ps.WorkDir, ps.Build.ArtifactName)
	if err := l.buildCache.Retrieve(ctx, buildcache.BuildObj, ps.Build.ArtifactName, executableDestination); err != nil {
		return "", err
	}

	return executableDestination, nil
}
