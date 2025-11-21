// Package git implements git related utilities
package nanogit

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	gitutil "github.com/grafana/grafana-bench/pkg/git/util"
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"
)

const (
	defaultBatch       = 100
	defaultConcurrency = 10
)

// gitRegRegexp matches the usual git reference patterns
var (
	gitRefRegexp = regexp.MustCompile(`^(heads/[^/]+|refs/heads/[^/]+|refs/tags/[^/]+)$`)

	commitHash = regexp.MustCompile(`^[a-f0-9]{7,40}$`)

	ErrRefNotFound = errors.New("reference not found")
)

type NanogitRepo struct {
	repo        string
	client      nanogit.Client
	batch       int
	concurrency int
}

// NewSource returns a new GitRepo instance.
func NewSource(
	repo string,
	token string,
) (*NanogitRepo, error) {
	var opts []options.Option
	if token != "" {
		// for tokens, gituser must be passed. Empty user is rejected
		opts = append(opts, options.WithBasicAuth("gituser", token))
	}

	client, err := nanogit.NewHTTPClient(repo, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating nanogit client %w", err)
	}

	return &NanogitRepo{
		repo:        repo,
		client:      client,
		batch:       defaultBatch,
		concurrency: defaultConcurrency,
	}, nil
}

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// Returns the revision that was retrieved
func (g *NanogitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	if revision == "" {
		return "", fmt.Errorf("revision must be provided")
	}

	err := gitutil.ValidateTargetDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("invalid target %w", err)
	}

	hash, err := g.resolveRevision(ctx, revision)
	if err != nil {
		return "", fmt.Errorf("resolving revision %w", err)
	}

	err = g.clone(ctx, targetDir, hash, checkoutDirs)
	if err != nil {
		return "", fmt.Errorf("cloning %s at %s to %s: %w", g.repo, revision, targetDir, err)
	}

	// return the short revision hash
	return hash.String()[:7], nil
}

func (g *NanogitRepo) resolveRevision(ctx context.Context, revision string) (hash.Hash, error) {
	switch {
	// it is already a commit hash
	case commitHash.MatchString(revision):
		if len(revision) != 40 {
			return hash.Hash{}, fmt.Errorf("a full 40-character commit hash is required"+
				", got %d characters: %s", len(revision), revision)
		}
		commitHash, err := hash.FromHex(revision)
		if err != nil {
			return hash.Hash{}, fmt.Errorf("parsing commit hash %s: %w", revision, err)
		}
		return commitHash, nil

	// if already a full reference, get it
	case gitRefRegexp.MatchString(revision):
		ref, err := g.client.GetRef(ctx, revision)
		if err != nil {
			return hash.Hash{}, fmt.Errorf("resolving revision %s: %w", revision, err)
		}
		return ref.Hash, nil
	// not a full reference, try usual reference patterns
	default:
		for _, prefix := range []string{"heads", "refs/heads", "refs/tags"} {
			ref, err := g.client.GetRef(ctx, fmt.Sprintf("%s/%s", prefix, revision))
			if err == nil {
				return ref.Hash, nil
			}

			var refNotFound *nanogit.RefNotFoundError
			if errors.As(err, &refNotFound) {
				continue
			}

			return hash.Hash{}, fmt.Errorf("retrieving ref %w", err)
		}

		return hash.Hash{}, fmt.Errorf("%w: %q", ErrRefNotFound, revision)
	}
}

func (g *NanogitRepo) clone(ctx context.Context, targetDir string, commitHash hash.Hash, checkoutDirs []string) error {

	// Prepare clone options
	cloneOpts := nanogit.CloneOptions{
		Path:        targetDir,
		Hash:        commitHash,
		BatchSize:   g.batch,
		Concurrency: g.concurrency,
	}

	// Handle checkout directories (path filtering)
	if len(checkoutDirs) > 0 {
		// Convert checkout directories to include paths with glob patterns
		includePaths := make([]string, len(checkoutDirs))
		for i, dir := range checkoutDirs {
			// Add /** to make it include all files under the directory
			includePaths[i] = dir + "/**"
		}
		cloneOpts.IncludePaths = includePaths
	}

	// Perform the clone
	_, err := g.client.Clone(ctx, cloneOpts)
	return err
}
