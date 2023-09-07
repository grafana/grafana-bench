package tester

import (
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type TestRun struct {
	*TesterService

	// relative path to file or folder to run in the test suite
	Tests string

	// git hash of the test suite
	SuiteRevision string

	// run a single iteration of k6 tests
	SmokeTest bool

	// report results to k6 cloud instead of locally
	ReportToK6Cloud bool
}

// DEPRECATED: kept for history/naming scheme if we want to bring it back
// ResultsDirectory gets the directory to output results for a specific run
// Takes the uuid identifier
// results/2023-06-21/12:00:00-6a50a8b1-8dbf-46ee-a179-86b2598ffeee/
//func (tr *TestRun) ResultsDirectory(provisionStateIdentifier string) string {
//  formattedTime := time.Now().Format("2006-01-02|15-04-05")
//  parts := strings.Split(formattedTime, "|")
//  date, time := parts[0], parts[1]

//  // 12:00:00-uuid
//  runIdentifier := fmt.Sprintf("%s-%s", time, provisionStateIdentifier)

//  return path.Join(tr.resultsDir, date, runIdentifier)
//}

// Resolves test suite for the test run
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to link the build suite to the service.
// This could probably be optimized for less checkouts/etc later
func (tr *TestRun) ResolveTestSuite() error {
	// clone repo if doesn't exist
	exists, _ := utils.PathExists(tr.TestSuiteDir)
	if !exists {
		log.Println("test-run: cloning build suite")

		if err := sh.RunV("git", "clone", tr.GrafanaTestRepo, tr.TestSuiteDir); err != nil {
			return fmt.Errorf("test-run: Error checking out grafana test repo %s", err)
		}
	}

	// update repo + checkout branch
	err := utils.DoInDir(tr.LocalDir, tr.TestSuiteDir, func() error {

		// if we don't specify a revision, assume we want to run exactly what is
		// there. e.g. local development
		if tr.SuiteRevision == "" {
			return nil
		}

		// check current branch. Don't do anything if it's the same as what is
		// currently there.
		// TODO: this logic may not make sense in scenarios where it's set to main
		// but someone hasn't checked out in a while and wants to update. That would
		// require a manual update. perhaps check the git sha. review later.
		cm := exec.Command("git", "branch", "--show-current")
		branch, err2 := cm.CombinedOutput()
		if err2 != nil {
			panic(fmt.Sprintf("error getting current branch %s", err2))
		}
		if strings.TrimSpace(string(branch)) == tr.SuiteRevision {
			return nil
		}

		// get latest main
		if err := utils.ExecStdout(exec.Command("git", "checkout", "main")); err != nil {
			return fmt.Errorf("test-run: Error checking out grafana test repo %s", err)
		}

		// update
		if err := utils.ExecStdout(exec.Command("git", "pull")); err != nil {
			return err
		}

		// checkout if not main
		var err error
		if tr.SuiteRevision != "main" {
			err = utils.ExecStdout(exec.Command("git", "checkout", tr.SuiteRevision))
			if err != nil {
				return err
			}
		}

		// get the commit hash
		cmd := exec.Command("git", "log", "-1", "--pretty=format:%H")
		hash, err := cmd.CombinedOutput()

		tr.SuiteRevision = string(hash)

		return err
	})

	return err
}

// GetTestSuiteFiles builds a list of k6 tests to run based on TestFiles
// environment variable. If the file has a js extension, we will try to run that
// file. If a directory is provided, we will list files in that directory using
// glob syntax and run each of those.
// e.g. TestFiles=dashboards will get all files in tests/tests/dashboards/**.*.js
// e.g. TestFiles=dashboards/dashboard_read.js will only run dashboard_read.js
func (tr *TestRun) GetTestSuiteFiles() ([]string, error) {
	// single file if we have .js extension
	if strings.Contains(tr.Tests, ".js") {
		// verify existence of absolute path
		p := path.Join(tr.TestSuiteDir, "tests", tr.Tests)
		exists, _ := utils.PathExists(p)
		if !exists {
			return []string{}, fmt.Errorf("test-run: File %s was not found. double check you passed the correct argument when creating test run", p)
		}
		return []string{p}, nil
	}

	d := path.Join(tr.TestSuiteDir, "tests", tr.Tests)
	exists, _ := utils.PathExists(d)
	if !exists {
		return []string{}, fmt.Errorf("test-run: Path %s was not found. double check you passed the correct argument when creating test run", d)
	}

	files, err := utils.GlobByExtension(d, ".js")
	if err != nil {
		panic(err)
	}

	return files, nil
}

// Gets test suite files with the path to file changed to remotePath
// <testSuiteDir>/tests/dashboards/dashboard_read.js to
// <remotePath>/tests/dashboards/dashboard_read.js
func (tr *TestRun) GetRemoteTestSuiteFiles(remotePath string) ([]string, error) {
	// get all the files
	files, err := tr.GetTestSuiteFiles()
	if err != nil {
		return []string{}, err
	}

	remoteFiles := []string{}
	for _, v := range files {
		relativeFile := strings.Replace(v, tr.TestSuiteDir, remotePath, -1)
		remoteFiles = append(remoteFiles, relativeFile)
	}

	return remoteFiles, nil
}

// BundleTestSuite bundles the test suite into a tarball
// TODO only ship lib and tests dir from suite directory
func (tr *TestRun) PrepareTestBundle(bundlePath string) error {
	log.Println("provisioner: compressing test bundle")
	return utils.CompressFolder(tr.TestSuiteDir, bundlePath)
}

// return the folder of the testfile relative to testSuite/tests/
// eg /home/k6/work/tests/suite/tests/dashboards/dashboard_create.js -> dashboards
// eg testSuite/tests/mytest.js -> ""
func (tr *TestRun) RelativeFolder(testFile string) string {
	p := strings.TrimPrefix(testFile, path.Join(tr.TestSuiteDir, "tests"))
	return path.Dir(p)
}

func (tr *TestRun) GetShortTestRevision() string {

	var suiteRevision string
	err := utils.DoInDir(tr.LocalDir, tr.TestSuiteDir, func() error {
		// get the commit hash
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
		hash, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}

		suiteRevision = strings.TrimSpace(string(hash))
		return nil
	})

	if err != nil {
		log.Println("test-run: Error getting short revision", err)
	}

	return suiteRevision
}
