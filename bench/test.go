package bench

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Run tests runs test on an instance of grafana already available at port 3000.
// This does not manage booting or stopping the instance
func (b *BenchRun) Test(ctx context.Context) error {
	if err := b.ResolveConfig(ctx); err != nil {
		return err
	}

	if err := b.ResolveTestSuite(ctx); err != nil {
		return err
	}

	// Wait for the server to start up
	waitForLiveGrafana()

	// run k6 tests
	err := utils.DoInDir(b.ProjectRoot, "tests", func() error {
		envVars := make(map[string]string)
		envVars["MACHINE_SPEC"] = getMachineSpec()
		envVars["TEST_SUITE_REVISION"] = b.TestSuiteRevision
		envVars["TEST_SUMMARY_DIR"] = b.TestSummaryDir

		tests, err := getTestSuiteFiles(b.ProjectRoot, b.TestSuite)
		if err != nil {
			return err
		}

		// run the tests
		for _, testFile := range tests {
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
	// provider, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|m1max|65536|10|3.2 GHz|arm64|darwin"
}
