package git

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"slices"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/grafana/grafana-bench/internal/testutils/gittest"
)

var repoFiles = map[string]string{
	"directory/file":          "file",
	"anotherDir/another_file": "another file",
}

func TestGitSource(t *testing.T) {
	// start a git server
	gitSrv, err := gittest.NewGitServer(
		t.Context(),
		gittest.GitServerConfig{
			RepoName: "bench",
			User:     "bench",
			Password: "benchbench",
			Email:    "bench-testing@grafana.com",
		})
	if err != nil {
		t.Fatalf("creating test git server %v", err)
	}

	// create a test git repository (will have a master branch by default)
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

	// add the remote
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitSrv.URL},
	})
	if err != nil {
		t.Fatalf("creating remote: %v", err)
	}

	// add files in repository
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

	// commit files (path must be relative to repo's root)
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

	// create tag 'v0.0.0'
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

	// create a branch 'test-branch'
	branchName := "test-branch"
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = wt.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: branchRef})
	if err != nil {
		t.Fatalf("creating branch: %v", err)
	}

	// push all to the remote
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/heads/*"), // Push all branches
			config.RefSpec("+refs/tags/*:refs/tags/*"),   // Push all tags
		},
		Auth: &http.BasicAuth{Username: "gituser", Password: gitSrv.Token},
	})


	testCases := []struct {
		name      string
		revision  string
		dirs      []string
		expectErr bool
	}{
		{
			name:      "get empty",
			revision:  "",
			expectErr: true,
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
			revision:  "00000aaaaabbbbbcccccdddddeeeeefffff11111",
			expectErr: true,
		},
		{
			name:      "short hash",
			revision:  "0abcdef",
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

		t.Run(tc.name, func(t *testing.T) {
			source, err := NewGitSource(gitSrv.URL, gitSrv.Token)
			if err != nil {
				t.Fatalf("creating git Source %v", err)
			}

			targetDir := path.Join(t.TempDir(), "repo")
			_, err = source.Get(context.TODO(), targetDir, tc.revision)
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
		source, err := NewGitSource(gitSrv.URL, gitSrv.Token)
		if err != nil {
			t.Fatalf("creating git Source %v", err)
		}

		_, err = source.Get(context.TODO(), clonedRepo, "")
		if err == nil {
			t.Fatalf("should have failed")
		}
	})

	t.Run("get directories", func(t *testing.T) {
		t.Skip("not implemented")
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
					t.Fatalf("creating git Source %v", err)
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
