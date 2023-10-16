package tester

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type TestRunType string

const (
	// single iteration, fail slow, returns with exit code
	Smoke TestRunType = "smoke"
	// xxx iterations, don't fail, report to k6 cloud
	Load TestRunType = "load"
)

func (trt TestRunType) String() string {
	switch trt {
	case Smoke:
		return "smoke"
	case Load:
		return "load"
	default:
		panic("Unknown TestRunType")
	}
}

// Gets the TestRunType from a string
func TestRunTypeFromString(trt string) TestRunType {
	trt = strings.ToLower(trt)
	switch trt {
	case "smoke":
		return Smoke
	case "load":
		return Load
	default:
		panic(fmt.Sprintf("invalid test run type %s", trt))
	}
}

type TestRun struct {
	*TesterService
	// Check, Smoke, Load. See TestRunType
	Type TestRunType
	// relative path to file or folder to run in the test suite
	Tests string
	// git hash of the test suite
	SuiteRevision string
}

// Resolves test suite for the test run
func (tr *TestRun) ResolveTestSuite() error {
	if tr.UseCompiledTests {
		// verify we have tests
		exists, err := utils.PathExists(tr.TestRoot)
		if err != nil || !exists {
			return fmt.Errorf("error checking compiled tests in %s. err: %w", tr.TestRoot, err)
		}
		return nil
	}

	// clone repo if doesn't exist
	exists, _ := utils.PathExists(tr.TestSuiteDir)
	if !exists {
		tr.Log.Info("cloning build suite")

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
			return fmt.Errorf("error getting current branch %w", err2)
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

		// build the tests
		cmd := exec.Command("make", "build")
		err = utils.ExecStdout(cmd)
		if err != nil {
			return fmt.Errorf("Error building test suite: %w", err)
		}

		// get the commit hash
		tr.SuiteRevision, err = tr.GetShortTestRevisionFromCompiled()
		if err != nil {
			return fmt.Errorf("Error getting version from compiled test suite")
		}

		return err
	})

	return err
}

// GetTestSuiteFiles builds a list of k6 tests to run based on tr.TestSuite
// If tr.TestSuite has a js extension run that single file otherwise assume it's
// a folder and glob all of the .js files in it recursively
// e.g.
// TestFiles=dashboards/dashboard_read.js will run dashboard_read.js
// TestSuite=dashboards will run all files in dist/dashboards/**.*.js
//
// If TestSuite is blank, assume we want to run everything in dist/**.*.js
func (tr *TestRun) GetTestSuiteFiles() ([]string, error) {
	// single file if we have .js extension
	if strings.Contains(tr.Tests, ".js") {
		// verify existence of absolute path
		p := path.Join(tr.TestRoot, tr.Tests)
		exists, _ := utils.PathExists(p)
		if !exists {
			return []string{}, fmt.Errorf("test-run: File %s was not found. double check you passed the correct argument when creating test run", p)
		}
		return []string{p}, nil
	}

	var d string
	if tr.Tests == "all" {
		d = tr.TestRoot
	} else {
		d = path.Join(tr.TestRoot, tr.Tests)
	}

	// folder if no extension
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

// read .version from dist/ folder in test repo
func (tr *TestRun) GetShortTestRevisionFromCompiled() (string, error) {
	bytes, err := os.ReadFile(tr.VersionFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
