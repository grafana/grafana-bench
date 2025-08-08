// Package git implements git related utilities
package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"
)

// gitRegRegexp matchs the usual git reference patterns
var (
	gitRefRegexp = regexp.MustCompile(`^(heads/[^/]+|refs/heads/[^/]+|refs/tags/[^/]+)$`)

	commitHash = regexp.MustCompile(`^[a-f0-9]{7,40}$`)

	ErrRefNotFound = errors.New("reference not found")
)

type GitRepo struct {
	client nanogit.Client
}

// NewGitSource returns a new GitRepo instance.
func NewGitSource(
	repo string,
	token string,
) (*GitRepo, error) {
	var opts []options.Option
	if token != "" {
		// for tokens, gituser must be passed. Empty user is rejected
		opts = append(opts, options.WithBasicAuth("gituser", token))
	}

	client, err := nanogit.NewHTTPClient(repo, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating nanogit client %w", err)
	}

	return &GitRepo{
		client: client,
	}, nil
}

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// Returns the revision that was retrieved
func (g *GitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	if revision == "" {
		return "", fmt.Errorf("revision must be provided")
	}

	if len(checkoutDirs) > 0 {
		return "", fmt.Errorf("sparse checkout is not yet implemented")
	}

	err := validateTargetDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("invalid target %w", err)
	}

	hash, err := g.checkoutRevision(ctx, targetDir,revision)
	if err != nil {
		return "", fmt.Errorf("getting work tree %w", err)
	}

	return hash, nil
}

func validateTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)

	// if not exists, it's fine, we will create it
	if os.IsNotExist(err) {
		return nil
	}

	// un expected error
	if err != nil {
		return fmt.Errorf("accessing target dir %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target must be a directory")
	}

	empty, err := isEmpty(targetDir)
	if err != nil {
		return err
	}

	if !empty {
		return fmt.Errorf("target dir must be empty")
	}

	return nil
}


func isEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
      	if err != nil {
              return false, fmt.Errorf("accessing directory %w", err)
      	}

	return len(files) == 0, nil
}

func (g *GitRepo) checkoutRevision(ctx context.Context, target string, revision string) (string, error) {
	if commitHash.MatchString(revision) {
		return "", fmt.Errorf("checkout of commit not supported")
	}

	ref, err := g.resolveRevision(ctx, revision)
	if err != nil {
		return "", err
	}

	err = g.downloadTree(ctx, target, ref.Hash)
	if err != nil {
		return "", err
	}

	// return short hash
	return ref.Hash.String()[:7], nil
}

func (g *GitRepo) resolveRevision(ctx context.Context, revision string) (nanogit.Ref, error) {
	// if already a full reference, nothing to do
	if gitRefRegexp.MatchString(revision) {
		return g.client.GetRef(ctx, revision)
	}

	// try usual reference patterns
	for _, prefix := range []string{"heads", "refs/heads", "refs/tags"} {
		ref, err := g.client.GetRef(ctx, fmt.Sprintf("%s/%s", prefix, revision))
		if err == nil {
			return ref, nil
		}

		var refNotFound *nanogit.RefNotFoundError
		if errors.As(err, &refNotFound) {
			continue
		}

		return nanogit.Ref{}, fmt.Errorf("retrieving ref %w", err)
	}

	return nanogit.Ref{}, fmt.Errorf("%w: %q", ErrRefNotFound, revision)
}

// recursively downloads content from the repo at a given reference's hash
func (g *GitRepo) downloadTree(ctx context.Context, target string, hash hash.Hash) error {
	var err error

	if err = os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("creating target dir %s: %w", target, err)
	}

	tree, err := g.client.GetTree(ctx, hash)
	if err != nil {
		return fmt.Errorf("retrieving tree at hash %s: %w", hash.String(), err)
	}

	for _, e := range tree.Entries {
		// if entry is a  directory
		if e.Mode & 0o40000 != 0 {
			if err = g.downloadTree(ctx, filepath.Join(target, e.Name), e.Hash); err != nil {
				return err
			}
			continue
		}

		content, err := g.client.GetBlob(ctx, e.Hash)
		if err != nil {
			return fmt.Errorf("downloading file %s: %w", e.Name, err)
		}
		err = os.WriteFile(filepath.Join(target, e.Name), content.Content, os.FileMode(e.Mode))
		if err != nil {
			return fmt.Errorf("writing file %s: %w", filepath.Join(target, e.Name), err)
		}
	}

	return nil
}