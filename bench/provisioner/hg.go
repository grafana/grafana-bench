package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

var _ ProvisionDriver = (*HGDriver)(nil)

type HGDriver struct{}

func NewHGDriver() *HGDriver {
	return &HGDriver{}
}

func (d *HGDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	log := log.With("provisioner", "hg")
	log.Info("running test suite")

	if err := tr.ResolveTestSuite(); err != nil {
		return fmt.Errorf("provisioner: %w", err)
	}

	// verify we have tests
	machineSpec, err := d.GetMachineSpec(ctx, ps)
	if err != nil {
		return err
	}

	envVars := map[string]string{
		"MACHINE_SPEC":                machineSpec,
		"TEST_SUITE_REVISION":         tr.SuiteRevision,
		"GT_URL":                      ps.GrafanaInstance.HttpsServiceAddress(),
		"GT_USERNAME":                 ps.GrafanaInstance.GrafanaUser,
		"GT_PASSWORD":                 ps.GrafanaInstance.GrafanaPassword,
		"K6_PROMETHEUS_RW_USERNAME":   os.Getenv("K6_PROMETHEUS_RW_USERNAME"),
		"K6_PROMETHEUS_RW_PASSWORD":   os.Getenv("K6_PROMETHEUS_RW_PASSWORD"),
		"K6_PROMETHEUS_RW_SERVER_URL": os.Getenv("K6_PROMETHEUS_RW_SERVER_URL"),
		"K6_CLOUD_TOKEN":              tr.K6CloudToken,
		"K6_CLOUD_PROJECT_ID":         tr.K6CloudProjectId,
		"K6_CLOUD_TRACES_ENABLED":     "true",
	}

	// run k6 tests
	err = utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
		tests, err := tr.GetTestSuiteFiles()
		if err != nil {
			return err
		}

		var (
			startTime     = time.Now()
			totalDuration float32
			anyFailures   = false
		)

		// run the tests
		for iteration, testFile := range tests {
			jsonFile := getJsonOutputFilename(testFile)
			scenarioName := getScenarioName(testFile)

			// set the scenario name so it's accessible from the test
			envVars["SCENARIO_NAME"] = scenarioName

			log.Info("running test file", "file", testFile)
			log.Info("output json to", "file", jsonFile)

			args := []string{"run", testFile,
				"--out", "json=" + jsonFile,
				"--tag", "SUITE_RUN=" + ps.Identifier}

			if tr.Type == tester.Smoke {
				args = append(args, "--iterations", "1", "--vus", "1", "--out", "cloud")
			}

			cmd := exec.Command("k6", args...)

			// set env vars
			for key, value := range envVars {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
			}

			buf := bytes.NewBuffer(nil)
			cmd.Stdout = io.MultiWriter(buf, os.Stderr)
			cmd.Stderr = os.Stderr

			// run command
			err = cmd.Run()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
					anyFailures = true
				}
				log.Info("error running k6 command", "error", err)
			}

			// scenario + testDuration will be in milliseconds
			td, err := parseDurationFromJsonFile(scenarioName, jsonFile)
			if err != nil {
				log.Info("error processing json file", "error", err)
			}

			// NOTE total duration will be the total time for all tests to run
			// including setup and teardown and does not include the rest of the
			// time for bench to run
			var id, url string
			totalDuration += td.TotalDuration
			if tr.Type == tester.Load {
				id, url, err = parseK6CloudIdentifiersFromCLIOutput(buf.Bytes())
				if err != nil {
					log.Warn("error parsing cloud run from K6 summary", "error", err)
				}
			}

			testIterations, err := parseIterationCountFromCLIOutput(buf.Bytes())
			if err != nil {
				log.Warn("error parsing iterations from k6 summary", "error", err)

			}

			// test complete log
			log.Info("testRun",
				"suiteRun", ps.Identifier,
				"scenarioName", scenarioName,
				"grafanaVersion", ps.Build.GrafanaRevision,
				"folder", tr.RelativeFolder(testFile),
				"testFile", path.Base(testFile),
				"order", strconv.Itoa(iteration+1),
				"iterations", testIterations,
				"k6CloudUrl", url,
				"k6CloudID", id,
				"setupDuration", prettyMS(td.SetupDuration),
				"scenarioDuration", prettyMS(td.ScenarioDuration),
				"teardownDuration", prettyMS(td.TeardownDuration),
				"totalDuration", prettyMS(td.TotalDuration),
				"exitCode", exitCode,
			)
		}

		log.Info("suiteRun",
			// TODO pass the trigger from argo. (Manual, CI / release channel)
			"testTrigger", "CI",
			"grafanaVersion", ps.Build.GrafanaRevision,
			"suiteRun", ps.Identifier,
			"buildVersion", ps.Build.GrafanaRevision,
			"startTime", startTime.Format(time.RFC3339),
			"duration", prettyMS(totalDuration),
			"machineInfo", machineSpec,
			"anyFailures", anyFailures,
		)

		return nil
	})

	log.Info("test suite finished")

	return err
}

// Hardcoded based on kubeconfig set when hacking this together.
// would be cool to get this from kubernetes directly
func (d *HGDriver) GetMachineSpec(ctx context.Context, ps *ProvisionState) (string, error) {
	// driver, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|Intel(R) Xeon(R)|512000|2|2.8 GHz|x86_64|linux", nil
}

// Blocking call that waits for grafana to become ready
func (d *HGDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaInstance.ServiceAddress())
}

// Provision not implemented for hosted grafana driver
func (d *HGDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {
	log.Info("provisioner: provision not implemented for hosted grafana driver")
	return NilFunc, nil
}

// Provision not implemented for hosted grafana driver. state is not written to
// disk
func (d *HGDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	exists, err := utils.PathExists(ps.LocalDir)
	if err != nil {
		log.Info("provisioner: error checking if provision state exists", err)
	}

	if !exists {
		log.Info("provisioner: state not written to disk. exiting")
		return nil
	}

	log.Info("removing state directory", "dir", ps.LocalDir)
	return utils.Rm(ps.LocalDir)
}
