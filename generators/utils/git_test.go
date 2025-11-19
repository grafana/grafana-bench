package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetShortCommitSHA(t *testing.T) {
	// Create a temporary directory for our test repository
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Create a test file and commit it
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Add the file to staging
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_, err = worktree.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	// Create a commit
	commit, err := worktree.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test GetShortCommitSHA
	shortSHA, err := GetShortCommitSHA(tempDir)
	if err != nil {
		t.Fatalf("GetShortCommitSHA failed: %v", err)
	}

	// Verify the short SHA is 7 characters
	if len(shortSHA) != 7 {
		t.Errorf("Expected short SHA to be 7 characters, got %d: %s", len(shortSHA), shortSHA)
	}

	// Verify it matches the beginning of the full commit hash
	fullSHA := commit.String()
	if fullSHA[:7] != shortSHA {
		t.Errorf("Short SHA %s doesn't match beginning of full SHA %s", shortSHA, fullSHA)
	}
}

func TestGetShortCommitSHA_NonGitDirectory(t *testing.T) {
	// Create a temporary directory that's not a git repository
	tempDir, err := os.MkdirTemp("", "non-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test that it returns an error for non-git directory
	_, err = GetShortCommitSHA(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

func TestGetShortCommitSHA_NonExistentDirectory(t *testing.T) {
	// Test with non-existent directory
	_, err := GetShortCommitSHA("/this/directory/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}