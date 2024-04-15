package compile

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const makefileContent = `
build:
	echo "building"
`

func Test_Compiler(t *testing.T) {
	
	logBuffer := bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(&logBuffer, nil))

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

	// 2. create the Makefile in "master" branch
	wt.Checkout(&git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: plumbing.NewBranchReferenceName("master"),
	})

	makeFile, err := os.OpenFile(path.Join(repoDir, "Makefile"), os.O_CREATE | os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("creating makefile: %v", err)
	}
	_, err = makeFile.Write([]byte(makefileContent))
	if err != nil {
		t.Fatalf("writing to makefile: %v", err)
	}
	makeFile.Close()

	// 3. commit the Makefile
	_, err = wt.Add("Makefile")
	if err != nil {
		t.Fatalf("adding makefile: %v", err)
	}

	commitHash, err := wt.Commit("add makefile", &git.CommitOptions{
		Author: &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
	})
	if err != nil {
		t.Fatalf("committing makefile: %v", err)
	}

	// 4. create a branch 'test-branch'
	branchName := "test-branch"
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = wt.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: branchRef} )
	if err != nil {
		t.Fatalf("creating branch: %v", err)
	}

	// 5. create a tag 'v0.0.0'
	tagName := "v0.0.0"
	_, err = repo.CreateTag(
		tagName,
		commitHash,
		&git.CreateTagOptions{
			Tagger: &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
			Message: "release v0.0.0",
		},
	)
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}

	// 6. clone locally (used to test reuse of already cloned repos)
	clonedRepo := t.TempDir()
	_, err = git.PlainClone(
		clonedRepo,
		false,
		&git.CloneOptions{
			URL:      repoDir,
		},
	)
	if err != nil {
		t.Fatalf("cloning repo %v", err)
	}

	// TODO: add test cases where the make command fails
	testCases := []struct{
		name      string
		repo      string
		dir       string
		revision  string
		force     bool
		expectErr bool
	}{
		{
			name:      "reuse existing repo",
			repo:      repoDir,
			dir:       clonedRepo,
			revision:  "master",
			force:     false,
			expectErr: false,
		},
		{
			name:      "invalid local repo (not a git repo)",
			repo:      "",
			dir:       t.TempDir(),
			revision:  "master",
			force:     false,
			expectErr: true,
		},
		{
			name:      "build master",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  "master",
			force:     false,
			expectErr: false,
		},
		{
			name:      "build default (master)",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  "",
			force:     false,
			expectErr: false,
		},
		{
			name:      "build test branch",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  branchName,
			force:     false,
			expectErr: false,
		},
		{
			name:      "build tag",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  tagName,
			force:     false,
			expectErr: false,
		},
		{
			name:      "build hash",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  commitHash.String(),
			force:     false,
			expectErr: false,
		},
		{
			name:      "build non-existing hash",
			repo:      repoDir,
			dir:       path.Join(t.TempDir(), "repo"),
			revision:  "abcdef",
			force:     false,
			expectErr: true,
		},
		{
			name:      "build non-existing branch",
			repo:      repoDir,
			dir:       path.Join(t.TempDir()),
			revision:  "fake-branch",
			force:     false,
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
			compiler := NewTestCompiler(
				log,
				tc.dir,
				tc.repo,
				tc.revision,
				tc.force,
			)

			err = compiler.CompileTestSuite(context.TODO())
			if err != nil && !tc.expectErr {
				t.Fatalf("compiling test: %v", err)
			}
		})
	} 

}

