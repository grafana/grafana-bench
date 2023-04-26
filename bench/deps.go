package bench

import (
	"fmt"

	"github.com/grafana/grafana-bench/bench/deps"
	"github.com/magefile/mage/sh"
)

// CheckDeps ensures that k6 is installed
func (b *Config) CheckDeps() error {
	// ensure k6 is installed
	if err := sh.Run("which", "k6"); err != nil {
		return fmt.Errorf("K6 not found. Install k6 for your platform. https://k6.io/docs/get-started/installation/")
	}

	return nil
}

// ResolveTestSuite ensures test suite is cloned locally and set to the correct
// version
func (b *Config) ResolveTestSuite() error {
	if err := deps.BootstrapTestSuite(b.ProjectRoot); err != nil {
		return err
	}

	testSuiteVersion, err := deps.ResolveTestSuite(b.ProjectRoot, b.TestSuiteVersion)
	if err != nil {
		return err
	}

	b.TestSuiteVersion = testSuiteVersion
	return nil
}

// ResolveBuildSuite ensures build suite is cloned locally
func (b *Config) ResolveBuildSuite() error {
	return deps.BootstrapBuildSuite(b.ProjectRoot)
}

// UpdateDeps updates local build and test suite repos
func (b *Config) UpdateDeps() error {
	fmt.Println("Updating build suite")
	if err := deps.UpdateBuildSuite(b.ProjectRoot); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Updating test suite")
	if err := deps.UpdateTestSuite(b.ProjectRoot); err != nil {
		return err
	}

	return nil
}
