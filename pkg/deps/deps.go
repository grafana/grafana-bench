package deps

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// Bootstrap clones test and build repos locally
func BootstrapDeps(projectRoot string) error {
	// ensure k6 is installed
	if err := sh.Run("which", "k6"); err != nil {
		return fmt.Errorf("K6 not found. Install k6 for your platform. https://k6.io/docs/get-started/installation/")
	}

	// get build suite
	if err := BootstrapBuildSuite(projectRoot); err != nil {
		return err
	}

	// get test suite
	if err := BootstrapTestSuite(projectRoot); err != nil {
		return err
	}

	return nil
}

// Update test and clone repos
func UpdateDeps(projectRoot string) error {
	if err := UpdateBuildSuite(projectRoot); err != nil {
		return err
	}

	if err := UpdateTestSuite(projectRoot); err != nil {
		return err
	}

	return nil
}
