// Package git implements git related utilities
package gogit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"


	gitutil "github.com/grafana/grafana-bench/pkg/git/util"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	gitRef = regexp.MustCompile(`^(heads/[^/]+|refs/heads/[^/]+|refs/tags/[^/]+)$`)

	commitHash = regexp.MustCompile(`^[a-f0-9]{7,40}$`)
)

type GitRepo struct {
	Lg        *slog.Logger
	Repo      string
	RepoToken string
}

// NewGitSource returns a new GitRepo instance.
func NewSource(
	repo string,
	token string,
) (*GitRepo, error) {
	return &GitRepo{
		Repo:      repo,
		RepoToken: token,
	}, nil
}

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// Returns the revision that was retrieved
func (tc *GitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	err := gitutil.ValidateTargetDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("invalid target %w", err)
	}

	repo, err := tc.clone(targetDir, revision)
	if err != nil {
		return "", err
	}

	hash, err := tc.resolveRef(repo, revision)
	if err != nil {
		return "", err
	}

	// Perform the checkout
	if err := tc.performCheckout(repo, hash, targetDir, checkoutDirs); err != nil {
		return "", fmt.Errorf("checking out revision %q: %w", revision, err)
	}

	// Return short revision hash
	return hash.String()[:7], nil
}

// getAuth returns HTTP auth method if token is available
func (tc *GitRepo) getAuth() http.AuthMethod {
	if tc.RepoToken != "" {
		return &http.BasicAuth{Username: "gituser", Password: tc.RepoToken}
	}
	return nil
}

// clone the repository at the given revision
func (tc *GitRepo) clone(targetDir string, revision string) (*git.Repository, error) {
	var (
		repo         *git.Repository
		err          error
	)

	switch {
	case revision == "":
		repo, err = tc.cloneDefaultBranch(targetDir)
		if err != nil {
			return nil, fmt.Errorf("cloning default branch from %s: %w", tc.Repo, err)
		}

	case commitHash.MatchString(revision):
		repo, err = tc.cloneFullRepository(targetDir)
		if err != nil {
			return nil, fmt.Errorf("cloning repo %s for commit %s: %w", tc.Repo, revision, err)
		}

	case gitRef.MatchString(revision):
		repo, err = tc.cloneRef(targetDir, revision)
		if err != nil {
			return nil, fmt.Errorf("cloning repo %s for ref %s: %w", tc.Repo, revision, err)
		}

	default:
		// is a short reference, try using common patters
		for _, ref := range []string{"refs/heads", "refs/tags"} {
			repo, err = tc.cloneRef(targetDir, fmt.Sprintf("%s/%s", ref, revision))
			if err == nil {
				return repo, nil
			}
			
			if !errors.Is(err, git.NoMatchingRefSpecError{}) {
				return nil, fmt.Errorf("cloning repo %s for ref %s: %w", tc.Repo, revision, err)
			}
		}

		// fall back
		repo, err = tc.cloneFullRepository(targetDir)
		if err != nil {
			return nil, fmt.Errorf("cloning repo %s: %w", tc.Repo, err)
		}
	}

	if repo == nil {
		return nil, fmt.Errorf("could not resolve revision %s in repository %s", revision, tc.Repo)
	}

	return repo, nil
}

// cloneDefaultBranch clones only the default branch with minimal data
func (tc *GitRepo) cloneDefaultBranch(targetDir string) (*git.Repository, error) {
	return git.PlainClone(targetDir, false, &git.CloneOptions{
		NoCheckout:   true,
		URL:          tc.Repo,
		Auth:         tc.getAuth(),
		Depth:        1,    // Only get the latest commit
		SingleBranch: true, // Only get the default branch
	})
}

// cloneBranch attempts to clone a specific branch with minimal data
func (tc *GitRepo) cloneRef(targetDir, ref string) (*git.Repository, error) {
	return git.PlainClone(targetDir, false, &git.CloneOptions{
		NoCheckout:    true,
		URL:           tc.Repo,
		Auth:          tc.getAuth(),
		Depth:         1,                                       // Only get the latest commit
		SingleBranch:  true,                                    // Only get this branch
		ReferenceName: plumbing.ReferenceName(ref),             // Specific branch
	})
}

// cloneFullRepository clones the complete repository with all history and references
// This is needed when we need to resolve arbitrary references that might not be available
// in shallow or single-branch clones (e.g., commit hashes, complex refs, etc.)
func (tc *GitRepo) cloneFullRepository(targetDir string) (*git.Repository, error) {
	repo, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		NoCheckout: true,
		URL:        tc.Repo,
		Auth:       tc.getAuth(),
		// No Depth or SingleBranch restrictions - get everything
	})
	if err != nil {
		return nil, err
	}

	// Fetch all remote references to ensure we have everything
	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
		Auth:     tc.getAuth(),
	})

	// Ignore "already up to date" errors
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("fetching all references: %w", err)
	}

	return repo, nil
}

// performCheckout handles the actual checkout with sparse directories
func (tc *GitRepo) performCheckout(repo *git.Repository, hash plumbing.Hash, targetDir string, checkoutDirs []string) error {
	tree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting work tree: %w", err)
	}

	checkoutOptions := &git.CheckoutOptions{
		Hash: hash,
	}

	// Only set sparse checkout if directories are specified
	if len(checkoutDirs) > 0 {
		checkoutOptions.SparseCheckoutDirectories = checkoutDirs
	}

	err = tree.Checkout(checkoutOptions)
	if err != nil {
		return fmt.Errorf("checking out: %w", err)
	}

	// Validate sparse checkout directories
	for _, dir := range checkoutDirs {
		if _, err := os.Stat(filepath.Join(targetDir, dir)); err != nil {
			return fmt.Errorf("directory not checked out: %q", dir)
		}
	}

	return nil
}

func (tc *GitRepo) resolveRef(repo *git.Repository, revision string) (plumbing.Hash, error) {
	if revision == "" {
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("getting current branch: %w", err)
		}
		return head.Hash(), nil
	}
	
	revisionHash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("resolving commit hash %q: %w", revision, err)
	}
	return *revisionHash, nil
}
