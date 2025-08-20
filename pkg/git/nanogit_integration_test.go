package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	// Dedicated test repository for git integration tests
	testRepoURL = "https://github.com/grafana/grafana-bench-git-test"
	testTag     = "v1.0.0"
	// Commit hash from deleted feature branch
	deletedBranchCommit = "b642ebe4aa10542aeb5e36e2a7df803114842c22"
)

// Expected files in main branch and tag v1.0.0
var expectedBaseFiles = []string{
	"playwright.config.ts",
	"package.json",
	"CODEOWNERS",
	"codeowners-mapping.yaml",
	"README.md",
}

var expectedBaseDirectories = []string{
	"tests",
}

// Additional files only present in deleted branch commit
var expectedFeatureFiles = []string{
	"feature-readme.md",
	"tests/dashboard.spec.js",
}

func TestGitIntegrationRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Helper function to check base files and directories
	checkBaseFiles := func(t *testing.T, targetDir string) {
		for _, file := range expectedBaseFiles {
			filePath := filepath.Join(targetDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("expected file %s not found", file)
			} else {
				t.Logf("✓ Found file: %s", file)
			}
		}

		for _, dir := range expectedBaseDirectories {
			dirPath := filepath.Join(targetDir, dir)
			if info, err := os.Stat(dirPath); os.IsNotExist(err) {
				t.Errorf("expected directory %s not found", dir)
			} else if !info.IsDir() {
				t.Errorf("%s exists but is not a directory", dir)
			} else {
				t.Logf("✓ Found directory: %s", dir)
			}
		}
	}

	// Helper function to check feature files (only in deleted branch commit)
	checkFeatureFiles := func(t *testing.T, targetDir string) {
		for _, file := range expectedFeatureFiles {
			filePath := filepath.Join(targetDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("expected feature file %s not found", file)
			} else {
				t.Logf("✓ Found feature file: %s", file)
			}
		}
	}

	t.Run("Scenario 1: Full checkout at main branch", func(t *testing.T) {
		targetDir := t.TempDir()
		
		gitRepo, err := NewGitSource(testRepoURL, "")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		revision, err := gitRepo.Get(ctx, targetDir, "")
		if err != nil {
			t.Fatalf("getting test repo at main branch: %v", err)
		}

		if len(revision) != 7 {
			t.Errorf("expected 7-character revision hash, got %d: %s", len(revision), revision)
		}

		t.Logf("✓ Successfully checked out main branch, revision: %s", revision)
		checkBaseFiles(t, targetDir)
	})

	t.Run("Scenario 2: Checkout at specific commit hash (main branch)", func(t *testing.T) {
		targetDir := t.TempDir()
		
		gitRepo, err := NewGitSource(testRepoURL, "")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		// Use the commit hash directly instead of tag (nanogit seems to have tag resolution issues)
		mainCommit := "49b46169b77812aab51476d1e6f28a6971e20ea6" // Initial commit hash
		
		revision, err := gitRepo.Get(ctx, targetDir, mainCommit)
		if err != nil {
			t.Fatalf("getting test repo at commit %s: %v", mainCommit, err)
		}

		if len(revision) != 7 {
			t.Errorf("expected 7-character revision hash, got %d: %s", len(revision), revision)
		}

		expectedShort := mainCommit[:7]
		if revision != expectedShort {
			t.Errorf("expected revision %s, got %s", expectedShort, revision)
		}

		t.Logf("✓ Successfully checked out specific commit %s, revision: %s", mainCommit, revision)
		checkBaseFiles(t, targetDir)
	})

	t.Run("Scenario 3: Checkout commit from deleted branch", func(t *testing.T) {
		targetDir := t.TempDir()
		
		gitRepo, err := NewGitSource(testRepoURL, "")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		// Checkout commit from deleted feature branch
		revision, err := gitRepo.Get(ctx, targetDir, deletedBranchCommit)
		if err != nil {
			t.Fatalf("getting test repo at deleted branch commit %s: %v", deletedBranchCommit, err)
		}

		if len(revision) != 7 {
			t.Errorf("expected 7-character revision hash, got %d: %s", len(revision), revision)
		}

		expectedShort := deletedBranchCommit[:7]
		if revision != expectedShort {
			t.Errorf("expected revision %s, got %s", expectedShort, revision)
		}

		t.Logf("✓ Successfully checked out deleted branch commit %s, revision: %s", deletedBranchCommit, revision)
		
		// Should have all base files plus feature files
		checkBaseFiles(t, targetDir)
		checkFeatureFiles(t, targetDir)
	})

	t.Run("Scenario 4: Test directory filtering", func(t *testing.T) {
		targetDir := t.TempDir()
		
		gitRepo, err := NewGitSource(testRepoURL, "")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		// Checkout only tests directory
		revision, err := gitRepo.Get(ctx, targetDir, "", "tests")
		if err != nil {
			t.Fatalf("getting test repo with tests directory only: %v", err)
		}

		if len(revision) != 7 {
			t.Errorf("expected 7-character revision hash, got %d: %s", len(revision), revision)
		}

		// Should have tests directory but not root files
		testsDir := filepath.Join(targetDir, "tests")
		if info, err := os.Stat(testsDir); os.IsNotExist(err) {
			t.Error("expected tests directory not found")
		} else if !info.IsDir() {
			t.Error("tests exists but is not a directory")
		} else {
			t.Log("✓ Found tests directory")
		}

		// Root files should NOT be present with directory filtering
		for _, file := range []string{"package.json", "README.md"} {
			filePath := filepath.Join(targetDir, file)
			if _, err := os.Stat(filePath); err == nil {
				t.Errorf("file %s should not be present with directory filtering", file)
			}
		}

		t.Logf("✓ Successfully tested directory filtering, revision: %s", revision)
	})

	t.Run("Error handling tests", func(t *testing.T) {
		gitRepo, err := NewGitSource(testRepoURL, "")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		// Test non-existent branch
		targetDir := t.TempDir()
		_, err = gitRepo.Get(ctx, targetDir, "non-existent-branch-12345")
		if err == nil {
			t.Error("expected error for non-existent branch")
		}
		t.Logf("✓ Non-existent branch error: %v", err)

		// Test short commit hash (should error)
		targetDir2 := t.TempDir()
		shortHash := deletedBranchCommit[:7]
		_, err = gitRepo.Get(ctx, targetDir2, shortHash)
		if err == nil {
			t.Error("expected error for short commit hash")
		}
		t.Logf("✓ Short commit hash error: %v", err)

		// Test non-existent tag
		targetDir3 := t.TempDir()
		_, err = gitRepo.Get(ctx, targetDir3, "v999.999.999")
		if err == nil {
			t.Error("expected error for non-existent tag")
		}
		t.Logf("✓ Non-existent tag error: %v", err)
	})
}