package tester

import (
	"log"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
)

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
	p := strings.TrimPrefix(testFile, tr.TestRoot)
	return path.Dir(p)
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
