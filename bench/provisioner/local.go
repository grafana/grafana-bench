package provisioner

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

var _ ProvisionDriver = (*LocalDriver)(nil)

type LocalDriver struct {
	buildCache *buildcache.BuildCache
}

func NewLocalDriver(buildCache *buildcache.BuildCache) *LocalDriver {
	return &LocalDriver{
		buildCache: buildCache,
	}
}

// Provision - provisions Grafana + test runner
func (l *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {

	// setup the directory structure
	executable, err := setupGrafanaWorkdir(ctx, l.buildCache, ps)
	if err != nil {
		return NilFunc, err
	}

	// boot grafana
	killFunc, err := boot(ctx, ps, executable)
	if err != nil {
		return NilFunc, err
	}

	// TODO figure out how to get this from ENV or custom.ini
	ps.GrafanaInstance = &VMInstance{
		IPAddress:   "localhost",
		ServicePort: "3000",
	}

	return killFunc, nil
}

// Blocking call that waits for grafana to become ready
func (l *LocalDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaInstance.ServiceAddress())
}

// Check - checks if Grafana + test runner are ready
func (l *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) bool {
	return IsLive(ps.GrafanaInstance.ServiceAddress())
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (l *LocalDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	// kill the process
	err := ps.killFunc()
	if err != nil {
		return err
	}

	fmt.Println("removing state directory:", ps.LocalDir)

	// remove the state directory
	return utils.Rm(ps.LocalDir)
}

// Runs tests against a provisioned instance of Grafana
func (l *LocalDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	// resolve test suite
	err := tr.ResolveTestSuite()
	if err != nil {
		return fmt.Errorf("provisioner: error running test suite: %w", err)
	}

	// run k6 tests
	err = utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
		resultsDir := tr.ResultsDirectory(ps.Identifier)
		err := os.MkdirAll(resultsDir, 0755)
		if err != nil {
			return err
		}

		envVars := map[string]string{
			"MACHINE_SPEC":        getMachineSpec(),
			"TEST_SUITE_REVISION": tr.SuiteRevision,
			"TEST_SUMMARY_DIR":    resultsDir,
			// set port number
			//GF_SERVER_HTTP_PORT=9191
		}

		if tr.ReportToK6Cloud {
			envVars["k6_CLOUD_TOKEN"] = tr.K6CloudToken
			//envVars["k6_CLOUD_PROJECT_ID"] = tr.K6CloudProjectID
		}

		tests, err := tr.GetTestSuiteFiles()
		if err != nil {
			return err
		}

		// run the tests
		for _, testFile := range tests {
			fmt.Println("provisioner: running test file:", testFile)

			var cmd *exec.Cmd
			if tr.ReportToK6Cloud {
				cmd = exec.Command("k6", "run", testFile, "-i", "1", "-u", "1", "-o", "cloud")
			} else {
				cmd = exec.Command("k6", "run", testFile, "-i", "1", "-u", "1")
			}

			// TODO figure out what to do with threshold errors from k6.
			// The ones in the test may not match what we need and will exist with
			// non-zero status code resulting in RunWithVar returning an error
			// an error even though we don't care about it. This isn't a GREAT
			// approach. We should figure out a way to tell k6 not to return an error
			// if threshold is breached rather than necessarily modifying the test

			// k6 run tests/tests/dashboards.js -i 1 -u 1 -o cloud
			_ = utils.ExecStdoutWithEnv(cmd, envVars)
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

// Gets machine spec for provisioned machine
// TODO IMPLEMENT ME
func getMachineSpec() string {
	// provider, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|m1max|65536|10|3.2 GHz|arm64|darwin"
}
