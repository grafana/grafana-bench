package provisioner

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

var _ ProvisionDriver = (*LocalDriver)(nil)

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

// Stubbed function to return when something goes wrong provisioning
func NilFunc() error {
	return nil
}

// Provision - provisions Grafana + test runner
func (l *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {

	// setup the directory structure
	executable, err := l.setupGrafanaWorkdir(ctx, ps)
	if err != nil {
		return NilFunc, err
	}

	// boot grafana
	killFunc, err := boot(ctx, ps, executable)
	if err != nil {
		return NilFunc, err
	}

	// setup test runner
	//err = setupTestWorkdir(ctx, ps)

	// Setup update provision state
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
	// TODO implement me
	return nil
}

// Runs tests against a provisioned instance of Grafana
func (l *LocalDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	// TODO implement me

	// resolve test suite
	err := tr.ResolveTestSuite()
	if err != nil {
		return fmt.Errorf("provisioner: error running test suite: %w", err)
	}

	// run k6 tests
	err = utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
		envVars := make(map[string]string)
		envVars["MACHINE_SPEC"] = getMachineSpec()
		envVars["TEST_SUITE_REVISION"] = tr.SuiteRevision
		envVars["TEST_SUMMARY_DIR"] = tr.SummaryDir

		tests, err := tr.GetTestSuiteFiles()
		if err != nil {
			return err
		}

		// run the tests
		for _, testFile := range tests {
			fmt.Println("provisioner: running test file:", testFile)

			// k6 run tests/tests/dashboards.js

			// TODO figure out how to ignore threshold errors from k6.
			// The ones in the test may not match what we need and will exist with
			// non-zero status code resulting in RunWithVar returning an error
			// an error even though we don't care about it. This isn't a GREAT
			// approach. We should figure out a way to tell k6 not to return an error
			// if threshold is breached rather than necessarily modifying the test
			_ = sh.RunWithV(envVars, "k6", "run", testFile, "-i", "1", "-u", "1")

			// TODO maybe stdout the location of the test file
		}

		return nil
	})

	return err
}

// Boots grafana on provisioned instance of Grafana
func boot(ctx context.Context, ps *ProvisionState, executable string) (func() error, error) {
	cmd := exec.Command(executable, "server")

	// function to return so we can kill the process
	killFunc := func() error {
		err := cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("provisioner: ERROR killing grafana PID: %w", err)
		}
		fmt.Println("provisioner: shutdown grafana pid ", cmd.Process.Pid)
		return nil
	}

	err := utils.DoInDir(utils.Getwd(), ps.WorkDir, func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})
	if err != nil {
		return NilFunc, err
	}

	return killFunc, nil
}

// Sets up directory with configs needed for testing a grafana
// build. This method expects the BuildArtifactPath to exist on disk.
func (l *LocalDriver) setupGrafanaWorkdir(ctx context.Context, ps *ProvisionState) (string, error) {
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

// Sets up directory and ensures repo is up to date
func setupTestWorkdir(ctx context.Context) error {
	return nil
}

// Gets machine spec for provisioned machine
// TODO IMPLEMENT ME
func getMachineSpec() string {
	// provider, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|m1max|65536|10|3.2 GHz|arm64|darwin"
}
