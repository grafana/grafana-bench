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
	ps.Log.Info("running test suite")

	if err := tr.ResolveTestSuite(); err != nil {
		return fmt.Errorf("provisioner: %w", err)
	}

	machineSpec, err := d.GetMachineSpec(ctx, ps)
	if err != nil {
		return err
	}

	if tr.Type == tester.Smoke {
		return smokeTest(ctx, ps, tr, machineSpec)
	} else {
		return loadTest(ctx, ps, tr, machineSpec)
	}
}

// Hardcoded based on kubeconfig set when hacking this together.
// would be cool to get this from kubernetes directly
func (d *HGDriver) GetMachineSpec(ctx context.Context, ps *ProvisionState) (string, error) {
	// driver, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|Intel(R) Xeon(R)|512000|2|2.8 GHz|x86_64|linux", nil
}

// Blocking call that waits for grafana to become ready
func (d *HGDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.Log, ps.GrafanaInstance.ServiceAddress())
}

// Provision not implemented for hosted grafana driver
func (d *HGDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {
	ps.Log.Warn("provision not implemented for provision driver")
	return NilFunc, nil
}

// Provision not implemented for hosted grafana driver. state is not written to
// disk
func (d *HGDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	ps.Log.Warn("destroy not implemented for provision driver")
	return nil
}

// load test runs 100 iteration(default specified in test suite) of each test
// specified and reports to k6 cloud
func loadTest(ctx context.Context, ps *ProvisionState, tr *tester.TestRun, machineSpec string) error {
	envVars := map[string]string{
		"MACHINE_SPEC":            machineSpec,
		"TEST_TYPE":               tr.Type.String(),
		"TEST_SUITE_REVISION":     tr.SuiteRevision,
		"GT_URL":                  ps.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":             ps.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":             ps.GrafanaInstance.ServicePassword,
		"K6_CLOUD_TOKEN":          tr.K6CloudToken,
		"K6_CLOUD_PROJECT_ID":     tr.K6CloudProjectId,
		"K6_CLOUD_TRACES_ENABLED": "true",
	}

	// run k6 tests
	return utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
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

			// build the command with buffer
			cmd, buf := prepareK6Command(ps.Identifier, testFile, jsonFile, envVars,
				"--out", "json="+jsonFile,
				"--out", "cloud",
			)

			// run command
			err = cmd.Run()
			exitCode := 0
			exitMessage := ""
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
					anyFailures = true
				}
				exitMessage = "error running k6 command: " + err.Error()
				ps.Log.Info("error running k6 command", "error", err)
			}

			// scenario + testDuration will be in milliseconds
			td, err := parseDurationFromJsonFile(ps.Log, scenarioName, jsonFile)
			if err != nil {
				ps.Log.Info("error processing json file", "error", err)
			}

			// NOTE total duration will be the total time for all tests to run
			// including setup and teardown and does not include the rest of the
			// time for bench to run
			var id, url string
			totalDuration += td.TotalDuration
			id, url, err = parseK6CloudIdentifiersFromCLIOutput(ps.Log, buf.Bytes())
			if err != nil {
				ps.Log.Warn("error parsing cloud run from K6 summary", "error", err)
			}

			testIterations, err := parseIterationCountFromCLIOutput(buf.Bytes())
			if err != nil {
				ps.Log.Warn("error parsing iterations from k6 summary", "error", err)
			}

			// test complete log
			ps.Log.Info("testRun",
				"benchVersion", ps.BenchRevision,
				"apiTestsVersion", tr.SuiteRevision,
				"testRun", newTestIdentifier(testFile, tr, ps.GrafanaBuildInfo.Revision),
				"suiteRun", ps.Identifier,
				"scenarioName", scenarioName,
				"grafanaUrl", ps.GrafanaInstance.Host,
				"grafanaVersion", ps.GrafanaBuildInfo.Revision,
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
				"exitMessage", exitMessage,
				"exitCode", exitCode,
			)
		}

		totalScenarioDurations := prettyMS(totalDuration)
		benchDuration := prettyMS(float32(time.Since(startTime).Milliseconds()))

		ps.Log.Info("suiteRun",
			"benchVersion", ps.BenchRevision,
			"apiTestsVersion", tr.SuiteRevision,
			"suiteRun", ps.Identifier,
			// TODO pass the trigger from argo. (Manual, CI / release channel)
			"testTrigger", "CI",
			"grafanaUrl", ps.GrafanaInstance.Host,
			"grafanaVersion", ps.GrafanaBuildInfo.Revision,
			"startTime", startTime.Format(time.RFC3339),
			"totalScenarioDurations", totalScenarioDurations,
			"duration", benchDuration,
			"machineInfo", machineSpec,
			"anyFailures", anyFailures,
		)

		return nil
	})
}

// smokeTest runs a single iteration of each test specified and does not report
// to k6 cloud
func smokeTest(ctx context.Context, ps *ProvisionState, tr *tester.TestRun, machineSpec string) error {
	envVars := map[string]string{
		"MACHINE_SPEC":        machineSpec,
		"TEST_TYPE":           tr.Type.String(),
		"TEST_SUITE_REVISION": tr.SuiteRevision,
		"GT_URL":              ps.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":         ps.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":         ps.GrafanaInstance.ServicePassword,
	}

	return utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
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

			// build the command with buffer
			cmd, buf := prepareK6Command(ps.Identifier, testFile, jsonFile, envVars)

			// run command
			err = cmd.Run()
			exitCode := 0
			exitMessage := ""
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
					anyFailures = true
				}
				exitMessage = "error running k6 command: " + err.Error()
				ps.Log.Info("error running k6 command", "error", err)
			}

			// get iterations
			testIterations, err := parseIterationCountFromCLIOutput(buf.Bytes())
			if err != nil {
				ps.Log.Warn("error parsing iterations from k6 summary", "error", err)
			}

			// scenario + testDuration will be in milliseconds
			td, err := parseDurationFromJsonFile(ps.Log, scenarioName, jsonFile)
			if err != nil {
				ps.Log.Info("error processing json file", "error", err)
			}
			// NOTE total duration will be the total time for all tests to run
			// including setup and teardown and does not include the rest of the
			// time for bench to run
			totalDuration += td.TotalDuration

			// test complete log
			ps.Log.Info("testRun",
				"benchVersion", ps.BenchRevision,
				"apiTestsVersion", tr.SuiteRevision,
				"testRun", newTestIdentifier(testFile, tr, ps.GrafanaBuildInfo.Revision),
				"suiteRun", ps.Identifier,
				"scenarioName", scenarioName,
				"grafanaUrl", ps.GrafanaInstance.Host,
				"grafanaVersion", ps.GrafanaBuildInfo.Revision,
				"folder", tr.RelativeFolder(testFile),
				"testFile", path.Base(testFile),
				"order", strconv.Itoa(iteration+1),
				"iterations", testIterations,
				"setupDuration", prettyMS(td.SetupDuration),
				"scenarioDuration", prettyMS(td.ScenarioDuration),
				"teardownDuration", prettyMS(td.TeardownDuration),
				"totalDuration", prettyMS(td.TotalDuration),
				"exitMessage", exitMessage,
				"exitCode", exitCode,
			)
		}

		totalScenarioDurations := prettyMS(totalDuration)
		benchDuration := prettyMS(float32(time.Since(startTime).Milliseconds()))

		ps.Log.Info("suiteRun",
			"benchVersion", ps.BenchRevision,
			"apiTestsVersion", tr.SuiteRevision,
			"suiteRun", ps.Identifier,
			// TODO pass the trigger from argo. (Manual, CI / release channel)
			"testTrigger", "CI",
			"grafanaUrl", ps.GrafanaInstance.Host,
			"grafanaVersion", ps.GrafanaBuildInfo.Revision,
			"startTime", startTime.Format(time.RFC3339),
			"totalScenarioDurations", totalScenarioDurations,
			"duration", benchDuration,
			"machineInfo", machineSpec,
			"anyFailures", anyFailures,
		)

		if anyFailures {
			dashboardUrl := fmt.Sprintf("https://ops.grafana-ops.net/d/d3381df1-fa32-4955-994a-e6a8bca58025/test-runs?var-SuiteRun=%s", ps.Identifier)
			return fmt.Errorf("Smoke test failed. Too many test failures. Review logs or see dashboard %s", dashboardUrl)
		}

		return nil
	})
}

// prepareK6Command builds the command with output set to standard output and a
// buffer and passes the cmd and buffer back to be executed and parsed
func prepareK6Command(identifier, testFile, jsonFile string, envVars map[string]string, args ...string) (*exec.Cmd, *bytes.Buffer) {
	defaultArgs := []string{
		"run",
		testFile,
		"--out", fmt.Sprintf(`json=%s`, jsonFile),
	}
	defaultArgs = append(defaultArgs, args...)

	cmd := exec.Command("k6", defaultArgs...)

	envVars["path"] = os.Getenv("PATH")
	envVars["K6_BROWSER_ENABLED"] = "true"
	envVars["K6_BROWSER_ARGS"] = "no-sandbox"
	// set env vars
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	buf := bytes.NewBuffer(nil)
	cmd.Stdout = io.MultiWriter(buf, os.Stderr)
	cmd.Stderr = os.Stderr

	return cmd, buf
}

// dashboardCreate.js-13:37:35-smoke-api-tests-cb5adc0-graf-10.2.0-60657
func newTestIdentifier(filename string, tr *tester.TestRun, grafanaVersion string) string {
	// {filename}-{time}-{type}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-%s-api-tests-%s-graf-%s",
		filename,
		time.Now().UTC().Format("15:04:05"),
		tr.Type.String(),
		tr.SuiteRevision,
		grafanaVersion,
	)
}
