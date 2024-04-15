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

func ImportSetupRepo(testingDir, testSuiteRepo string, log *slog.Logger) error {
	log.Info("importing test suite", "testSuiteRepo", testSuiteRepo, "targetDir", testingDir)

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

	// update repo + checkout branch
	err := ExecuteInDir(testingDir, func() error {
		// add a config in the repo with setup instructions
		// installCmd := exec.Command("yarn", "install")
		// if err := ExecStdout(installCmd); err != nil {
		// 	return fmt.Errorf("installing packages: %w", err)
		// }

		// installPlaywrightCmd := exec.Command("yarn", "playwright:install")
		// if err := ExecStdout(installPlaywrightCmd); err != nil {
		// 	return fmt.Errorf("installing playwright browsers: %w", err)
		// }

		return nil
	})

	return err
}
