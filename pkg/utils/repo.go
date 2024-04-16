package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
)

func GetTestingDirectory(targetDir, testSuiteRepo string) string {
	repoName := strings.Split(testSuiteRepo, ":")[1]
	testingDir := targetDir + "/" + repoName

	return testingDir
}

func CloneRepo(testingDir, testSuiteRepo string, log *slog.Logger) error {
	// clone repo if doesn't exist
	exists, _ := PathExists(testingDir)
	if exists {
		_, err := git.PlainOpen(testingDir)
		if err != nil {
			return fmt.Errorf("opening repo %s: %w", testSuiteRepo, err)
		}

	} else {
		log.Info("cloning test suite")
		_, err := git.PlainClone(
			testingDir,
			false,
			&git.CloneOptions{
				SingleBranch: true,
				Depth:        1,
				Auth:         nil,
				URL:          testSuiteRepo,
				Progress:     os.Stdout,
			},
		)

		if err != nil {
			return fmt.Errorf("error checking out test suite repo %s: %w", testSuiteRepo, err)
		}
		log.Info("cloning test suite complete")
	}

	return nil
}
