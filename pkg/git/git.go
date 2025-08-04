// Package git implements git related utilities
package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type GitRepo struct {
	Lg        *slog.Logger
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

// isCommitHash checks if the revision looks like a git commit hash
func isCommitHash(revision string) bool {
	// Match 7-40 character hex strings (typical git hash range)
	matched, _ := regexp.MatchString("^[a-f0-9]{7,40}$", revision)
	return matched
}

// getAuth returns HTTP auth method if token is available
func (tc *GitRepo) getAuth() http.AuthMethod {
	if tc.RepoToken != "" {
		return &http.BasicAuth{Username: "gituser", Password: tc.RepoToken}
	}
	return nil
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

// cloneSpecificBranch attempts to clone a specific branch with minimal data
func (tc *GitRepo) cloneSpecificBranch(targetDir, branch string) (*git.Repository, error) {
	return git.PlainClone(targetDir, false, &git.CloneOptions{
		NoCheckout:    true,
		URL:           tc.Repo,
		Auth:          tc.getAuth(),
		Depth:         1,                                       // Only get the latest commit
		SingleBranch:  true,                                    // Only get this branch
		ReferenceName: plumbing.NewBranchReferenceName(branch), // Specific branch
	})
}

// cloneForCommitHash clones with enough data to resolve arbitrary commits
// Note: go-git doesn't support partial clones yet, so we need full history for arbitrary commits
func (tc *GitRepo) cloneForCommitHash(targetDir string) (*git.Repository, error) {
	return git.PlainClone(targetDir, false, &git.CloneOptions{
		NoCheckout: true,
		URL:        tc.Repo,
		Auth:       tc.getAuth(),
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

// fetchSpecificRef tries to fetch only the needed reference
func (tc *GitRepo) fetchSpecificRef(repo *git.Repository, revision string) error {
	auth := tc.getAuth()

	// Try as branch first
	branchRefSpec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", revision, revision))
	err := repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{branchRefSpec},
		Auth:     auth,
		Depth:    1, // Only get the tip
	})

	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}

	// Try as tag
	tagRefSpec := config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", revision, revision))
	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{tagRefSpec},
		Auth:     auth,
		Depth:    1,
	})

	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}

	// Fallback: fetch all refs (current behavior)
	return repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
		Auth:     auth,
	})
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

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// Returns the revision that was retrieved
func (tc *GitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	var (
		repo         *git.Repository
		err          error
		checkoutHash plumbing.Hash
	)

	switch {
	case revision == "":
		repo, err = tc.cloneDefaultBranch(targetDir)
		if err != nil {
			return "", fmt.Errorf("cloning default branch from %s: %w", tc.Repo, err)
		}

		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("getting current branch: %w", err)
		}
		checkoutHash = head.Hash()

	case isCommitHash(revision):
		repo, err = tc.cloneFullRepository(targetDir)
		if err != nil {
			return "", fmt.Errorf("cloning repo for commit %s: %w", tc.Repo, err)
		}

		revisionHash, err := repo.ResolveRevision(plumbing.Revision(revision))
		if err != nil {
			return "", fmt.Errorf("resolving commit hash %q: %w", revision, err)
		}
		checkoutHash = *revisionHash

	default:
		repo, err = tc.cloneSpecificBranch(targetDir, revision)

		if err != nil {
			// Fallback: need full repo to resolve arbitrary references/commits
			repo, err = tc.cloneFullRepository(targetDir)
			if err != nil {
				return "", fmt.Errorf("cloning repo %s: %w", tc.Repo, err)
			}
		}

		// Check if we're already on the right branch
		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("getting current branch: %w", err)
		}

		if head.Name().Short() == revision {
			checkoutHash = head.Hash()
		} else {
			// Resolve the revision to a hash
			revisionHash, err := repo.ResolveRevision(plumbing.Revision(revision))
			if err != nil {
				return "", fmt.Errorf("resolving reference %q: %w", revision, err)
			}
			checkoutHash = *revisionHash
		}
	}

	// Perform the checkout
	if err := tc.performCheckout(repo, checkoutHash, targetDir, checkoutDirs); err != nil {
		return "", fmt.Errorf("checking out revision %q: %w", revision, err)
	}

	// Return short revision hash
	return checkoutHash.String()[:7], nil
}
