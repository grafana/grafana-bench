package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
	// resolve test suite to correct version etc
	if err := os.MkdirAll(filepath.Join("work", "test", "suite"), os.FileMode(0755)); err != nil {
		return fmt.Errorf("provisioner: %w", err)
	}

	//err := tr.ResolveTestSuite()
	//if err != nil {
	//  return fmt.Errorf("provisioner: error running test suite: %w", err)
	//}

	// run k6 tests
	err := utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
		resultsDir := tr.ResultsDirectory(ps.Identifier)
		err := os.MkdirAll(resultsDir, 0755)
		if err != nil {
			return err
		}

		machineSpec, err := d.GetMachineSpec(ctx, ps)
		if err != nil {
			return err
		}

		envVars := map[string]string{
			"MACHINE_SPEC":        machineSpec,
			"TEST_SUITE_REVISION": tr.SuiteRevision,
			"TEST_SUMMARY_DIR":    resultsDir,
			"GT_URL":              ps.GrafanaInstance.HttpsServiceAddress(),
			"GT_USERNAME":         ps.GrafanaInstance.GrafanaUser,
			"GT_PASSWORD":         ps.GrafanaInstance.GrafanaPassword,
		}

		if tr.ReportToK6Cloud {
			envVars["K6_CLOUD_TOKEN"] = strings.TrimSpace(tr.K6CloudToken)
			envVars["K6_CLOUD_PROJECT_ID"] = strings.TrimSpace(tr.K6CloudProjectId)
		}

		tests, err := tr.GetTestSuiteFiles()
		if err != nil {
			return err
		}

		var (
			startTime     = time.Now()
			totalDuration int
		)

		// run the tests
		for iteration, testFile := range tests {
			jsonFile := getJsonOutputFilename(testFile)
			scenarioName := getScenarioName(testFile)

			log.Info("running test file", "file", testFile)
			log.Info("output json to", "file", jsonFile)

			cmd := exec.Command("k6", "run", testFile,
				"--out", "cloud",
				"--out", "json="+jsonFile,
				"--tag", "SUITE_RUN="+ps.Identifier)

			// set env vars
			for key, value := range envVars {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
			}

			buf := bytes.NewBuffer(nil)
			cmd.Stdout = io.MultiWriter(buf, os.Stderr)
			cmd.Stderr = os.Stderr

			// run command
			err = cmd.Run()
			if err != nil {
				log.Info("error running k6 command", "error", err)
			}

			// scenario + testDuration will be in milliseconds
			scenarioDuration, testDuration, err := processTestJson(scenarioName, jsonFile)
			if err != nil {
				log.Info("error processing json file", "error", err)
			}

			// build log output
			m := map[string]string{
				"suiteRun":   ps.Identifier,
				"folder":     tr.RelativeFolder(testFile),
				"testFile":   path.Base(testFile),
				"order":      strconv.Itoa(iteration + 1),
				"k6CloudUrl": getCloudRunURL(buf.Bytes()),
				// maybe do something to make this value pretty for the dashboard.
				"scenarioDuration": strconv.Itoa(scenarioDuration),
				"totalDuration":    strconv.Itoa(testDuration),
			}

			// NOTE total duration will be the total time for all tests to run
			// including setup and teardown and does not include the rest of the
			// time for bench to run
			totalDuration += testDuration

			// test complete log
			log.Info(fmt.Sprintf("%+v", m))
		}

		// suite complete log
		// {test trigger, time of day, machine info, build version, duration??}
		m := map[string]string{
			// TODO figure out how to pass the trigger from argo
			"testTrigger":  "manual",
			"suiteRun":     ps.Identifier,
			"buildVersion": ps.Build.GrafanaRevision,
			"startTime":    startTime.Format(time.RFC3339),
			"duration":     strconv.Itoa(totalDuration),
			"machineInfo":  machineSpec,
		}
		log.Info(fmt.Sprintf("%+v", m))

		return nil
	})

	return err
}

// this is hardcoded based on kubeconfig set when hacking this together. would be cool to get this from kubernetes directly
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

// pattern to match the cloud output url inside parenthesis
//
//	output: cloud (https://jefflevinslunch.grafana.net/a/k6-app/runs/1876021), json (/tmp/dashboard_create.json)
var k6CloudOutputPattern = regexp.MustCompile(`output:\s*cloud\s*\(([^)]+)\)`)

// getCloudRunURL extracts the cloud run url from the k6 command output
func getCloudRunURL(b []byte) string {
	// Find the first match of the pattern in the input
	match := k6CloudOutputPattern.FindSubmatch(b)

	if len(match) >= 2 {
		// The URL is captured in the second element of the match
		url := string(match[1])
		log.Info("Found k6 cloud output url", "url", url)
		return url
	} else {
		log.Info("URL not found")
		return ""
	}
}

// dashboard_create.js -> /tmp/dashboard_create.json
func getJsonOutputFilename(filename string) string {
	jsonName := filepath.Base(filename)
	jsonName = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
	return path.Join("/tmp", jsonName+".json")
}

// we expect scenarios to be named like the file
// tests/dashboards/dashboard_create.js -> dashboardCreate
func getScenarioName(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(filename, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

// TODO implement me
func processTestJson(scenarioName, jsonFile string) (int, int, error) {
	scenarioDuration := 0
	totalDuration := 0

	// parse file
	// get total duration
	// get scenario duration

	return scenarioDuration, totalDuration, nil
}
