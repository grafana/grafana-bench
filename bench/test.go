package bench

import (
	"fmt"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Run tests runs test on an instance of grafana already available at port 3000.
// This does not manage booting or stopping the instance
func (b *Config) Test() error {
	// get featureToggles & buildInfo from response /api/frontend/settings
	// only contains list of things that are turned on
	liveConfig, err := getLiveConfig()
	if err != nil {
		fmt.Println("error getting live config from booted grafana:", err)
	} else {
		fmt.Println(liveConfig)
	}

	// run k6 tests
	err = utils.DoInDir(b.ProjectRoot, "tests", func() error {

		envVars := make(map[string]string)
		envVars["machineSpec"] = getMachineSpec()

		// TODO:
		// 1. start here. see if we can get these environment variables passed in
		// correctly
		// 2. then verify environment variables and test suite information output
		// 3. change references of "commit" when referring to test suite or build
		// suite and use "revision"
		// 4. start thinking about collection results from test suite

		tests := getTestSuiteFiles()
		for _, testFile := range tests {
			// k6 run tests/tests/dashboards.js
			if err := sh.RunWithV(envVars, "k6", "run", testFile); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// GetTestSuiteFiles builds a list of k6 tests to run based on TestFiles
// environment variable. If the file has an extension, we will try to run that
// file. If a directory is provided, we will list files in that directory using
// glob syntax and run each of those.
// e.g. TestFiles=dashboards will get all files in tests/tests/dashboards/**.*.js
// e.g. TestFiles=dashboards/dashboard_read.js will only run dashboard_read.js
func getTestSuiteFiles() []string {
	tests := []string{"tests/dashboards/dashboard_create.js"}
	return tests
}

func getMachineSpec() string {
	return "local|m1max|65536/3.2 GHz|arm64|darwin"
}
