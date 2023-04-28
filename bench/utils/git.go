package utils

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// determines when string is a branch or commit
func IsCommitHash(branchOrCommit string) bool {
	return len(branchOrCommit) == 40
}

// getLatestBranchCommit gets latest commit from a given branch including main
func GetLatestBranchCommit(repo, branch string) (string, error) {
	if branch == "main" {
		branch = "HEAD"
	}

	resolved, err := sh.Output("git", "ls-remote", repo, branch, "-c7")
	if err != nil {
		return "", fmt.Errorf("Error resolving git commit %s: %s", branch, err)
	}

	// get first column
	// e0b2aeffa34ba6ca812ff3db6a08adee7a89b6d4        HEAD
	commit := strings.Split(resolved, "\t")[0]

	if commit == "" {
		return "", fmt.Errorf("Branch: %s could not be resolve. verify it is a full git ref or branch name", branch)
	}

	return commit, nil
}
