package bench

import (
	"fmt"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Run tests runs test on an instance of grafana already available at port 3000.
// This does not manage booting or stopping the instance
func (b *Config) Test() error {
	if err := b.ResolveConfig(); err != nil {
		return err
	}

	if err := b.ResolveTestSuite(); err != nil {
		return err
	}

	// Wait for the server to start up
	waitForLiveGrafana()

	// get featureToggles & buildInfo from response /api/frontend/settings
	// only contains list of things that are turned on
	//liveConfig, err := getLiveConfig()
	//if err != nil {
	//  fmt.Println("error getting live config from booted grafana:", err)
	//} else {
	//  fmt.Println(liveConfig)
	//}

	// run k6 tests
	err := utils.DoInDir(b.ProjectRoot, "tests", func() error {

		envVars := make(map[string]string)
		envVars["MACHINE_SPEC"] = getMachineSpec()
		envVars["TEST_SUITE_REVISION"] = b.TestSuiteRevision

		// TODO: START HERE
		// 1. work on getTestSuiteFiles
		// 2. start thinking about collection results from test suite
		// file output should come from an env variable but have a default for all
		// tests

		tests, err := getTestSuiteFiles(b.ProjectRoot, b.TestSuite)
		if err != nil {
			return err
		}

		// run the tests
		for _, testFile := range tests {
			// k6 run tests/tests/dashboards.js

			// FIXME ignore errors on this as thresholds on k6 may not match and will throw
			// an error even though we don't care about it. This isn't a GREAT
			// approach. We should figure out a way to tell k6 not to return an error
			// if threshold is breached rather than necessarily modifying the test
			_ = sh.RunWithV(envVars, "k6", "run", testFile, "-i", "1", "-u", "1")

			// collect results
		}

		return nil
	})

	return err
}

// TODO IMPLEMENT ME
// GetTestSuiteFiles builds a list of k6 tests to run based on TestFiles
// environment variable. If the file has an extension, we will try to run that
// file. If a directory is provided, we will list files in that directory using
// glob syntax and run each of those.
// e.g. TestFiles=dashboards will get all files in tests/tests/dashboards/**.*.js
// e.g. TestFiles=dashboards/dashboard_read.js will only run dashboard_read.js
//
// TODO further investigate using k6 scenarios - https://k6.io/docs/using-k6/scenarios/
func getTestSuiteFiles(projectRoot, testSuite string) ([]string, error) {
	// default to dashboards test suite
	if testSuite == "" {
		testSuite = "dashboards"
	}

	// single file if we have .js extension
	if strings.Contains(testSuite, ".js") {
		// verify existence of absolute path
		p := path.Join(projectRoot, "tests", "tests", testSuite)
		exists, _ := utils.PathExists(p)
		if !exists {
			return []string{}, fmt.Errorf("File %s was not found", p)
		}
		return []string{p}, nil
	}

	d := path.Join(projectRoot, "tests", "tests", testSuite)
	exists, _ := utils.PathExists(d)
	if !exists {
		return []string{}, fmt.Errorf("Path %s was not found", d)
	}

	files, err := utils.Glob(d, ".js")
	if err != nil {
		panic(err)
	}

	return files, nil
}

// TODO IMPLEMENT ME
func getMachineSpec() string {
	return "local|m1max|65536|3.2 GHz|arm64|darwin"
}
