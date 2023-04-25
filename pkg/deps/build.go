package deps

import (
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/pkg/utils"
	"github.com/magefile/mage/sh"
)

// BootstrapBuildSuite downloads build suite locally
func BootstrapBuildSuite(projectRoot string) error {
	// check if build repo cloned locally
	exists, err := utils.PathExists(path.Join(projectRoot, "build"))
	if err != nil {
		return fmt.Errorf("Issue checking directory path")
	}

	if !exists {
		if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-build", "build"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}
	}

	return nil

}

// UpdateBuildSuite updates build suite repo
func UpdateBuildSuite(projectRoot string) error {
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
	return err
}
