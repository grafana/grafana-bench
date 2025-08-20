package git

import (
	"context"
	"fmt"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)


var repoFiles = map[string]string{
	"directory/file": "file",
	"anotherDir/another_file": "another file",
}

func TestGitSource(t *testing.T) {
	// test setup

	// 1. create a test git repository (will have a master branch by default)
	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("initializing repository: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("getting work tree: %v", err)
	}

	wt.Checkout(&git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: plumbing.NewBranchReferenceName("master"),
	})

	// 2 create files in repository
	for path, content := range repoFiles {
		path = filepath.Join(repoDir, path)
		err = os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			t.Fatalf("creating directory in repository: %v", err)
		}
		err = os.WriteFile(path, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("copying files to repository: %v", err)
		}
	}

	// 3. commit files (path must be relative to repo's root)
	_, err = wt.Add(".")
	if err != nil {
		t.Fatalf("adding files to commit: %v", err)
	}

	commitHash, err := wt.Commit("add test files", &git.CommitOptions{
		Author: &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
	})
	if err != nil {
		t.Fatalf("committing files: %v", err)
	}

	// 4. create a branch 'test-branch'
	branchName := "test-branch"
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = wt.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: branchRef})
	if err != nil {
		t.Fatalf("creating branch: %v", err)
	}

	// 5. create a tag 'v0.0.0'
	tagName := "v0.0.0"
	_, err = repo.CreateTag(
		tagName,
		commitHash,
		&git.CreateTagOptions{
			Tagger:  &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
			Message: "release v0.0.0",
		},
	)
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}

	// Start a git HTTP server using git http-backend
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("cannot find git: %v", err)
	}

	gitHandler := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		Env: []string{
			fmt.Sprintf("GIT_PROJECT_ROOT=%s", repoDir),
			"GIT_HTTP_EXPORT_ALL=true",
		},
	}

	gitSrv := httptest.NewServer(gitHandler)
	defer gitSrv.Close()

	testCases := []struct {
		name      string
		revision  string
		dirs      []string
		expectErr bool
	}{
		{
			name:      "get default",
			revision:  "",
			expectErr: false,
		},
		{
			name:      "get master",
			revision:  "master",
			expectErr: false,
		},
		{
			name:      "get branch",
			revision:  branchName,
			expectErr: false,
		},
		{
			name:      "get tag",
			revision:  tagName,
			expectErr: false,
		},
		{
			name:      "get hash",
			revision:  commitHash.String(),
			expectErr: false,
		},
		{
			name:      "get non-existing hash",
			revision:  "abcdef",
			expectErr: true,
		},
		{
			name:      "get non-existing branch",
			revision:  "fake-branch",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		// always start in "master" branch
		wt.Checkout(&git.CheckoutOptions{
			Create: false,
			Force:  false,
			Branch: plumbing.NewBranchReferenceName("master"),
		})

		t.Run(tc.name, func(t *testing.T) {
			source, err := NewGitSource(gitSrv.URL,"")
			if err != nil {
				t.Fatalf("creating git source: %v", err)
			}

			targetDir := path.Join(t.TempDir(), "repo")
			_, err = source.Get(context.TODO(), targetDir,tc.revision)
			if err != nil && !tc.expectErr {
				t.Fatalf("getting source: %v", err)
			}
		})
	}

	t.Run("get into an existing repository", func(t *testing.T) {
		clonedRepo := filepath.Join(t.TempDir(), "repo")
		_, err = git.PlainClone(
			clonedRepo,
			false,
			&git.CloneOptions{
				URL: repoDir,
			},
		)
		if err != nil {
			t.Fatalf("cloning repo %v", err)
		}

		// get source again into cloned repository
		source, err := NewGitSource(gitSrv.URL,"")
		if err != nil {
			t.Fatalf("creating git source: %v", err)
		}

		_, err = source.Get(context.TODO(), clonedRepo, "")
		if err == nil {
			t.Fatalf("should have failed")
		}
	})

	t.Run("get directories", func(t *testing.T) {
		testCases := []struct {
			title     string
			dirs      []string
			expectErr bool
		}{
			{
				title:     "checkout dir",
				dirs:      []string{"directory"},
				expectErr: false,
			},
			{
				title:     "checkout non-existing dir",
				dirs:      []string{"not-existing-dir"},
				expectErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.title, func(t *testing.T) {
				// must be in lexicographical order for slices.Equal to work
				dirs := tc.dirs
				slices.Sort(dirs)

				targetRepo := path.Join(t.TempDir(), "repo")
				source, err := NewGitSource(gitSrv.URL, "")
				if err != nil {
					t.Fatalf("creating git source: %v", err)
				}

				_, err = source.Get(context.TODO(), targetRepo, "", dirs...)
				if err != nil && !tc.expectErr {
					t.Fatalf("getting source: %v", err)
				}

				if tc.expectErr {
					return
				}

				// collect directories in cloned repo. Exclude .git
				checkedOutDirs := []string{}
				entries, _ := os.ReadDir(targetRepo)
				for _, e := range entries {
					if e.IsDir() && e.Name() != ".git" {
						checkedOutDirs = append(checkedOutDirs, e.Name())
					}
				}

				// check only selected directories were checked out
				if !slices.Equal(dirs, checkedOutDirs) {
					t.Fatalf("expected %v got %v", dirs, checkedOutDirs)
				}
			})
		}
	})
}
