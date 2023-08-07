package provisioner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

var _ ProvisionDriver = (*HGDriver)(nil)

type HGDriver struct{}

func NewHGDriver() *HGDriver {
	return &HGDriver{}
}

func (d *HGDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	// TODO
	// make sure test dir exists or clone

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

		// START HERE
		// Ship test suite run to k6 cloud
		// {test trigger, time of day, machine info, build version, duration??}

		// run the tests
		for _, testFile := range tests {

			jsonName := filepath.Base(testFile)
			jsonName = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
			jsonFile := path.Join("/tmp", jsonName+".json")

			fmt.Println("provisioner: running test file:", testFile)
			fmt.Println("provisioner: output json to:", jsonFile)

			cmd := exec.Command("k6", "run", testFile,
				"--out", "cloud",
				"--out", "json="+jsonFile,
				"--tag", "SUITE_RUN="+ps.Identifier)

			// set env vars
			for key, value := range envVars {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
			}

			// run command
			bytes, _ := cmd.CombinedOutput()
			fmt.Println(string(bytes))

			// get cloud run url and annotate json data
			cloudOutputUrl := getCloudRunURL(bytes)
			processedData, err := processTestJson(cloudOutputUrl, jsonFile)
			if err != nil {
				panic(err)
			}

			err = shipToLoki(processedData)
			if err != nil {
				panic(err)
			}
		}

		return nil
	})

	// TODO maybe ship a finish time to loki?

	return err
}

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
	fmt.Println("provisioner: provision not implemented for hosted grafana driver")
	return NilFunc, nil
}

// Provision not implemented for hosted grafana driver. state is not written to
// disk
func (d *HGDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	exists, err := utils.PathExists(ps.LocalDir)
	if err != nil {
		fmt.Println("provisioner: error checking if provision state exists", err)
	}

	if !exists {
		fmt.Println("provisioner: state not written to disk. exiting")
		return nil
	}

	fmt.Println("removing state directory:", ps.LocalDir)
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
		fmt.Println("provisioner: Found k6 cloud output url:", url)
		return url
	} else {
		fmt.Println("provisioner: URL not found.")
		return ""
	}
}

// TODO implement me
func processTestJson(cloudRunUrl, jsonFile string) (any, error) {
	// url:{url}, iterations:{iterations}, testFolder:{folder}, testName:
	// {testName}, duration:{duration in seconds}
	return nil, nil
}

func shipToLoki(data any) error {
	// TODO implement me
	return nil
}
