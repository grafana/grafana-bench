// Package git implements git related utilities
package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type GitRepo struct {
	Lg       *slog.Logger
	Repo      string
	RepoToken string
}

// NewGitSource returns a new GitRepo instance.
func NewGitSource(
	repo string,
	token string,
) *GitRepo {
	return &GitRepo{
		Repo:      repo,
		RepoToken: token,
	}
}

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// Returns the revision that was retrieved
func (tc *GitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	var (
		repo *git.Repository
		err  error
		auth http.AuthMethod
	)

	if tc.RepoToken != "" {
		// the user is required, but not used. Any non-empty value is accepted (!?)
		auth = &http.BasicAuth{Username: "gituser", Password: tc.RepoToken}
	}
	repo, err = git.PlainClone(
		targetDir,
		false,
		&git.CloneOptions{
			URL:  tc.Repo,
			Auth: auth,
		},
	)

	if err != nil {
		return "", fmt.Errorf("checking out repo %s: %w", tc.Repo, err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("error getting current branch %w", err)
	}

	checkoutHash := head.Hash()

	if revision != "" {
		// if we are not in the requested branch
		if head.Name().Short() != revision {
			// fetch remote refs and make them appear as local refs
			// assumes this is a cloned repository with an 'origin' remote
			err = repo.Fetch(&git.FetchOptions{
				RefSpecs: []config.RefSpec{"refs/*:refs/*"},
				Auth:     auth,
			})
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return "", fmt.Errorf("fetching references %w", err)
			}

			revisionHash, err := repo.ResolveRevision(plumbing.Revision(revision))
			if err != nil {
				return "", fmt.Errorf("resolving reference to revision %q :%w", revision, err)
			}
			// ResolveRevision returns &plumbing.Hash
			checkoutHash = *revisionHash
		}
	}

	tree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting work tree %w", err)
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Hash:                      checkoutHash,
		SparseCheckoutDirectories: checkoutDirs,
	})
	if err != nil {
		return "", fmt.Errorf("checking out revision %q: %w", revision, err)
	}

	// if CheckoutDirs was specified, check they are present, git won't report missing dirs
	for _, dir := range checkoutDirs {
		_, err := os.Stat(filepath.Join(targetDir, dir))
		if err != nil {
			return "", fmt.Errorf("directory not checked out: %q", dir)
		}
	}

	// set short revision
	revisionHash := checkoutHash.String()[:7]

	return revisionHash, nil
}
