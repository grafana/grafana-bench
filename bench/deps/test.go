package deps

import (
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// BootstrapTestSuite downloads build suite locally
func BootstrapTestSuite(projectRoot string) error {
	// check if test repo is cloned locally
	exists, err := utils.PathExists(path.Join(projectRoot, "tests"))
	if err != nil {
		return fmt.Errorf("Issue checking directory path")
	}

	if !exists {
		fmt.Println("Test suite not found. Cloning repo https://github.com/grafana/grafana-api-tests")
		if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-api-tests", "tests"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}
	}
	return nil
}

// UpdateTestSuite updates test suite repo
func UpdateTestSuite(projectRoot string) error {
	// tests
	err := utils.DoInDir(projectRoot, "tests", func() error {
		if err := sh.RunV("git", "checkout", "main"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}

		if err := sh.RunV("git", "pull"); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ResolveTestSuite ensures we have the correct version of the test suite at
// runtime. Leaving blank will get the sha of the repo as is. testSuiteVersion
// should be a branch name or sha that can be checked out
func ResolveTestSuite(projectRoot, testSuiteVersion string) (string, error) {

	// if no test suite provided, get the commit sha from the existing folder
	if testSuiteVersion == "" {
		err := utils.DoInDir(projectRoot, "tests", func() error {

			// get the sha
			sha, err := sh.Output("git", "log", "-n", "1", "--pretty=format:%H")
			if err != nil {
				return err
			}

			//assign to testSuiteVersion
			testSuiteVersion = sha
			return nil
		})

		// exit if we didn't have any errors
		if err == nil {
			return testSuiteVersion, nil
		}
	}

	// assume testSuiteVersion is a sha, or branch
	err := utils.DoInDir(projectRoot, "tests", func() error {
		// TODO add some sort of caching so we don't need to do this all the time
		// update repo
		if err := sh.Run("git", "pull"); err != nil {
			return err
		}

		// check out specific version of the test suite
		// we always check out otherwise we would need to exec another command to
		// get the current branch. this is fast
		if err := sh.Run("git", "checkout", testSuiteVersion); err != nil {
			return err

		}

		return nil
	})

	return testSuiteVersion, err
}
