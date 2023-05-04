package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/magefile/mage/sh"
)

// response for commit info
type GithubCommitResponse struct {
	Sha string `json:"sha"`
}

// response for branch info
type GithubBranchResponse struct {
	Commit struct {
		Sha string `json:"sha"`
	} `json:"commit"`
}

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

// ResolveCommitSha uses github api to get the full length commit sha from a
// shorter sha
func ResolveCommitFullSha(repo, commit string) (string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, commit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	gr := &GithubCommitResponse{}

	err = json.Unmarshal(body, gr)
	if err != nil {
		return "", err
	}

	return gr.Sha, nil
}

// ResolveLatestBranchCommit uses github api to return the latest commit sha on
// that branch
func ResolveLatestBranchCommit(repo, commit string) (string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s", repo, commit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	gr := &GithubBranchResponse{}

	err = json.Unmarshal(body, gr)
	if err != nil {
		return "", err
	}

	return gr.Commit.Sha, nil
}
