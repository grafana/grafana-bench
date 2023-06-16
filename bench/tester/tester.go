package tester

import (
	"fmt"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
)

type TestRun struct {
	*TesterService

	// folder in the test suite to run
	testSuite string

	// git hash of the test suite
	SuiteRevision string

	// output directory for test results
	SummaryDir string
}

// GetTestSuiteFiles builds a list of k6 tests to run based on TestFiles
// environment variable. If the file has an extension, we will try to run that
// file. If a directory is provided, we will list files in that directory using
// glob syntax and run each of those.
// e.g. TestFiles=dashboards will get all files in tests/tests/dashboards/**.*.js
// e.g. TestFiles=dashboards/dashboard_read.js will only run dashboard_read.js
//
// TODO further investigate using k6 scenarios - https://k6.io/docs/using-k6/scenarios/
func (tr *TestRun) GetTestSuiteFiles() ([]string, error) {
	// default to dashboards test suite
	if tr.testSuite == "" {
		tr.testSuite = "dashboards"
	}

	// single file if we have .js extension
	if strings.Contains(tr.testSuite, ".js") {
		// verify existence of absolute path
		p := path.Join(tr.TestSuiteDir, "tests", tr.testSuite)
		exists, _ := utils.PathExists(p)
		if !exists {
			return []string{}, fmt.Errorf("tester: File %s was not found", p)
		}
		return []string{p}, nil
	}

	d := path.Join(tr.TestSuiteDir, "tests", tr.testSuite)
	exists, _ := utils.PathExists(d)
	if !exists {
		return []string{}, fmt.Errorf("tester: Path %s was not found", d)
	}

	files, err := utils.GlobByExtension(d, ".js")
	if err != nil {
		panic(err)
	}

	return files, nil
}
