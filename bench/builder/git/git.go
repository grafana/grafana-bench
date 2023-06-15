package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ResolveCommitSha uses github api to get the full length commit sha from a
// shorter sha
func ResolveFullCommit(repo, commit string) (string, error) {
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

	body, err := io.ReadAll(resp.Body)
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

	body, err := io.ReadAll(resp.Body)
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
