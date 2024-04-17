package utils

import (
	"strings"
)

func GetTestingDirectory(targetDir, testSuiteRepo string) string {
	repoName := strings.Split(testSuiteRepo, ":")[1]
	testingDir := targetDir + "/" + repoName

	return testingDir
}
