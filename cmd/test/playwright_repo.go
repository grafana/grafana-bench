package test

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/grafana/grafana-bench/pkg/utils"
)

func ImportSetupRepo(targetDir, testSuiteRepo string, log *slog.Logger) error {
	var (
		repo *git.Repository
		err  error
	)

	log.Info("importing test suite", "testSuiteRepo", testSuiteRepo, "targetDir", targetDir)

	// clone repo if doesn't exist
	exists, _ := utils.PathExists(targetDir)
	if exists {
		repo, err = git.PlainOpen(targetDir)
		if err != nil {
			return fmt.Errorf("opening repo %s: %w", testSuiteRepo, err)
		}

	} else {
		log.Info("cloning test suite")
		repo, err = git.PlainClone(
			targetDir,
			false,
			&git.CloneOptions{
				Auth:     nil,
				URL:      testSuiteRepo,
				Progress: os.Stdout,
			},
		)

		if err != nil {
			return fmt.Errorf("error checking out test suite repo %s: %w", testSuiteRepo, err)
		}
	}

	println("repo: ", repo)
	// update repo + checkout branch
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current work directory %w", err)
	}

	// build the tests
	err = utils.DoInDir(workDir, targetDir, func() error {
		// add a config in the repo with setup instructions
		installCmd := exec.Command("yarn", "install")
		if err := utils.ExecStdout(installCmd); err != nil {
			return fmt.Errorf("installing packages: %w", err)
		}

		installPlaywrightCmd := exec.Command("yarn", "playwright:install")
		if err := utils.ExecStdout(installPlaywrightCmd); err != nil {
			return fmt.Errorf("installing playwright browsers: %w", err)
		}

		return nil
	})

	return err
}
