package tester

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type TestRun struct {
	*TesterService

	// relative path to file or folder to run in the test suite
	Tests string

	// git hash of the test suite
	SuiteRevision string

	// report results to k6 cloud instead of locally
	ReportToK6Cloud bool
}

// ResultsDirectory gets the directory to output results for a specific run
// Takes the uuid identifier
// results/2023-06-21/12:00:00-6a50a8b1-8dbf-46ee-a179-86b2598ffeee/
func (tr *TestRun) ResultsDirectory(provisionStateIdentifier string) string {
	formattedTime := time.Now().Format("2006-01-02|15-04-05")
	parts := strings.Split(formattedTime, "|")
	date, time := parts[0], parts[1]

	// 12:00:00-uuid
	runIdentifier := fmt.Sprintf("%s-%s", time, provisionStateIdentifier)

	return path.Join(tr.resultsDir, date, runIdentifier)
}

// Resolves test suite for the test run
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to link the build suite to the service.
// This could probably be optimized for less checkouts/etc later
func (tr *TestRun) ResolveTestSuite() error {
	// clone repo if doesn't exist
	exists, _ := utils.PathExists(tr.TestSuiteDir)
	if !exists {
		fmt.Println("test-run: cloning build suite")
		if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-api-tests", tr.TestSuiteDir); err != nil {
			return fmt.Errorf("test-run: Error checking out grafana test repo %s", err)
		}
	}

	// update repo + checkout branch
	err := utils.DoInDir(tr.LocalDir, tr.TestSuiteDir, func() error {
		if err := sh.RunV("git", "checkout", "main"); err != nil {
			return fmt.Errorf("test-run: Error checking out grafana test repo %s", err)
		}

		if err := sh.RunV("git", "pull"); err != nil {
			return err
		}

		// checkout if not main
		if tr.SuiteRevision != "main" {
			return sh.RunV("git", "checkout", tr.SuiteRevision)
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
func (tr *TestRun) GetTestSuiteFiles() ([]string, error) {
	// default to dashboards test suite
	if tr.Tests == "" {
		tr.Tests = "dashboards"
	}

	// single file if we have .js extension
	if strings.Contains(tr.Tests, ".js") {
		// verify existence of absolute path
		p := path.Join(tr.TestSuiteDir, "tests", tr.Tests)
		exists, _ := utils.PathExists(p)
		if !exists {
			return []string{}, fmt.Errorf("test-run: File %s was not found", p)
		}
		return []string{p}, nil
	}

	d := path.Join(tr.TestSuiteDir, "tests", tr.Tests)
	exists, _ := utils.PathExists(d)
	if !exists {
		return []string{}, fmt.Errorf("test-run: Path %s was not found", d)
	}

	files, err := utils.GlobByExtension(d, ".js")
	if err != nil {
		panic(err)
	}

	return files, nil
}
