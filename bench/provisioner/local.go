package provisioner

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
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

	fmt.Println(utils.Getwd())
	fmt.Println(ps.WorkDir)

	err = utils.DoInDir(utils.Getwd(), ps.WorkDir, func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})

	if err != nil {
		return killFunc, err
	}

	// TODO figure out how to get this info
	ps.GrafanaAddress = "localhost:3000"

	return killFunc, nil
}

func (l *LocalDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaAddress)
}

// Check - checks if Grafana + test runner are ready
func (l *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) bool {
	return IsLive(ps.GrafanaAddress)
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (l *LocalDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	return nil
}

// setupWorkdir sets up directory with configs needed for testing a grafana
// build. This method expects the BuildArtifactPath to exist on disk.
func (l *LocalDriver) setupWorkdir(ctx context.Context, ps *ProvisionState) (string, error) {
	// verify build artifact exists in the buildcache
	resolved, err := l.buildCache.Resolve(ctx, buildcache.TypeBuild, ps.Build.ArtifactBuildName)
	if err != nil {
		return "", fmt.Errorf("build-cache: error checking for grafana executable : %w", err)
	}
	if !resolved {
		return "", fmt.Errorf("build-cache: grafana executable not found: %s", ps.Build.ArtifactBuildName)
	}

	// verify defaults.ini exists in the buildcache
	resolved, err = l.buildCache.Resolve(ctx, buildcache.TypeINI, ps.Build.ArtifactININame)
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
	if err := l.buildCache.Retrieve(ctx, buildcache.TypeBuild, ps.Build.ArtifactBuildName, executableDestination); err != nil {
		return "", err
	}

	// copy defaults.ini into work dir
	iniDestination := path.Join(ps.WorkDir, "conf", "defaults.ini")
	if err := l.buildCache.Retrieve(ctx, buildcache.TypeINI, ps.Build.ArtifactININame, iniDestination); err != nil {
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
