package gittest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type TestRepo struct {
	user     string
	email    string
	workDir  string
	repo     *git.Repository
	workTree *git.Worktree
	remote   *GitServer
	head     plumbing.ReferenceName
	URL      string
	Token    string
}

func SetupTestRepo(ctx context.Context, workDir string) (*TestRepo, error) {
	// start a git server
	user := "bench"
	repoName := "bench"
	email := fmt.Sprintf("%s@testrepo.mail", user)
	remote, err := NewGitServer(
		ctx,
		GitServerConfig{
			RepoName: repoName,
			User:     user,
			Password: "test",
			Email:    email,
		})
	if err != nil {
		return nil, fmt.Errorf("creating test git server %w", err)
	}

	repo, err := git.PlainInit(workDir, false)
	if err != nil {
		return nil, fmt.Errorf("initializing repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting work tree: %w", err)
	}

	mainBranch := plumbing.NewBranchReferenceName("main")
	wt.Checkout(&git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: mainBranch,
	})

	// add the remote
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remote.URL},
	})
	if err != nil {
		return nil, fmt.Errorf("creating remote: %v", err)
	}

	return &TestRepo{
		URL:      remote.URL,
		Token:    remote.Token,
		user:     user,
		email:    email,
		repo:     repo,
		workTree: wt,
		head:     mainBranch,
		workDir:  workDir,
		remote:   remote,
	}, nil
}

func (r *TestRepo) Commit(message string, files map[string][]byte) (string, error) {
	// add files in repository
	for path, content := range files {
		fullPath := filepath.Join(r.workDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		if err != nil {
			return "", fmt.Errorf("creating directory in repository: %w", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			return "", fmt.Errorf("copying files to repository: %w", err)
		}

		_, err = r.workTree.Add(path)
		if err != nil {
			return "", fmt.Errorf("adding file to commit: %w", err)
		}
	}

	commitHash, err := r.workTree.Commit("add test files", &git.CommitOptions{
		Author: &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
	})
	if err != nil {
		return "", fmt.Errorf("committing files: %w", err)
	}

	// push commit to current head
	err = r.Push()
	if err != nil {
		return "", fmt.Errorf("pushing changes: %w", err)
	}

	return commitHash.String(), nil
}

func (r TestRepo) Tag(tag string, hash string) error {
	_, err := r.repo.CreateTag(
		tag,
		plumbing.NewHash(hash),
		&git.CreateTagOptions{
			Tagger:  &object.Signature{Name: r.user, Email: r.email},
			Message: "release v0.0.0",
		},
	)
	if err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	return r.Push()
}

func (r *TestRepo) CreateBranch(branch string) error {
	err := r.workTree.Checkout(&git.CheckoutOptions{
		Create: true,
		Force:  false,
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	return r.Push()
}

func (r *TestRepo) Push() error {
	// push all to the remote
	err := r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/heads/*"), // Push all branches
			config.RefSpec("+refs/tags/*:refs/tags/*"),   // Push all tags
		},
		Auth: &http.BasicAuth{Username: "gituser", Password: r.remote.Token},
	})

	if err != nil {
		return fmt.Errorf("pushing repo: %v", err)
	}

	return nil
}
