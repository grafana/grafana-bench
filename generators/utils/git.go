package utils

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// semVerRegex matches semantic version with optional v prefix. eg: v1.1.1
var semVerRegex = regexp.MustCompile(`v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)

// GetLatestBenchTag gets the latest semantic version tag from the repo.
// This is used for getting the latest tag for bench when updating docs or generating libsonnet.
func GetLatestBenchTag(repoPath string) (string, error) {
	// Open the repository
	repo, err := git.PlainOpenWithOptions(
		repoPath,
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return "", err
	}

	// Get all tags
	tagsIter, err := repo.Tags()
	if err != nil {
		return "", err
	}

	// Create a slice to store tag information
	type tagInfo struct {
		Name string
		Time time.Time
	}
	var tags []tagInfo

	// Collect all tags with their commit time
	err = tagsIter.ForEach(func(ref *plumbing.Reference) error {
		// Get the tag object
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			// If it's an annotated tag, use its timestamp
			tags = append(tags, tagInfo{
				Name: ref.Name().Short(),
				Time: tagObj.Tagger.When,
			})
			return nil
		}

		// For lightweight tags, get the commit it points to
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			// If we can't get the commit, just use the current time as a fallback
			tags = append(tags, tagInfo{
				Name: ref.Name().Short(),
				Time: time.Now(),
			})
			return nil
		}

		tags = append(tags, tagInfo{
			Name: ref.Name().Short(),
			Time: commit.Committer.When,
		})
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repository")
	}

	// Sort tags by time, newest first
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Time.After(tags[j].Time)
	})

	// get the latest tag that matches semver
	for _, t := range tags {
		if semVerRegex.MatchString(t.Name) {
			return t.Name, nil
		}
	}

	// Return error if no semantic version tags found
	return "", fmt.Errorf("no tags that meet semantic versioning standards found in repo")
}

// GetShortCommitSHA returns the short SHA of the current git commit
func GetShortCommitSHA(repoPath string) (string, error) {
	repo, err := git.PlainOpenWithOptions(
		repoPath,
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Get short SHA (first 7 characters)
	shortSHA := ref.Hash().String()[:7]
	return shortSHA, nil
}

// ErrFoundCommit is returned when we find the commit we're looking for
var ErrFoundCommit = errors.New("found commit")

// GetLatestCommitForFile returns the full SHA of the latest commit that modified a specific file
func GetLatestCommitForFile(repoPath string, filePath string) (string, error) {
	repo, err := git.PlainOpenWithOptions(
		repoPath,
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Get commit history
	commitIter, err := repo.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get commit history: %w", err)
	}

	// Find the latest commit that modified the specific file
	var latestSHA string
	err = commitIter.ForEach(func(commit *object.Commit) error {
		// Get the file stats for this commit
		stats, err := commit.Stats()
		if err != nil {
			// If we can't get stats, skip this commit
			return nil
		}

		// Check if the file was modified in this commit
		for _, stat := range stats {
			if stat.Name == filePath {
				latestSHA = commit.Hash.String()
				return ErrFoundCommit // Found what we're looking for
			}
		}

		return nil
	})

	if err != nil && err != ErrFoundCommit {
		return "", fmt.Errorf("failed to iterate commit history: %w", err)
	}

	if latestSHA == "" {
		return "", fmt.Errorf("no commits found that modified file %s", filePath)
	}

	return latestSHA, nil
}

// GetLatestCommitForFileOnMain returns the full SHA of the latest commit on the main branch that modified a specific file
func GetLatestCommitForFileOnMain(repoPath string, filePath string) (string, error) {
	repo, err := git.PlainOpenWithOptions(
		repoPath,
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	// Try to get the main branch reference
	mainRef, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		// If we can't find main branch (e.g., in CI with detached HEAD),
		// try to get the remote main branch
		mainRef, err = repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
		if err != nil {
			// As a fallback, use current HEAD (this might happen in shallow clones)
			mainRef, err = repo.Head()
			if err != nil {
				return "", fmt.Errorf("failed to get main branch reference or HEAD: %w", err)
			}
		}
	}

	// Get commit history from main branch
	commitIter, err := repo.Log(&git.LogOptions{
		From: mainRef.Hash(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get commit history from main: %w", err)
	}

	// Find the latest commit that modified the specific file on main
	var latestSHA string
	err = commitIter.ForEach(func(commit *object.Commit) error {
		// Get the file stats for this commit
		stats, err := commit.Stats()
		if err != nil {
			// If we can't get stats, skip this commit
			return nil
		}

		// Check if the file was modified in this commit
		for _, stat := range stats {
			if stat.Name == filePath {
				latestSHA = commit.Hash.String()
				return ErrFoundCommit // Found what we're looking for
			}
		}

		return nil
	})

	if err != nil && err != ErrFoundCommit {
		return "", fmt.Errorf("failed to iterate commit history on main: %w", err)
	}

	if latestSHA == "" {
		return "", fmt.Errorf("no commits found on main branch that modified file %s", filePath)
	}

	return latestSHA, nil
}