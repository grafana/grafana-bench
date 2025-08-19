// Package git provides a nanogit-based implementation with the same interface as the go-git based implementation
package git

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"
)

// NanoGitRepo implements the same interface as GitRepo but uses nanogit internally for better performance
type NanoGitRepo struct {
	Lg        *slog.Logger
	Repo      string
	RepoToken string
	client    nanogit.Client
}

// NewNanoGitSource returns a new NanoGitRepo instance that's compatible with the existing GitRepo interface
func NewNanoGitSource(repo string, token string) (*NanoGitRepo, error) {
	// Create nanogit HTTP client with authentication options
	var clientOptions []options.Option
	
	if token != "" {
		// Use Basic Auth with token as password to match go-git behavior
		// This is the standard pattern for GitHub/GitLab tokens
		clientOptions = append(clientOptions, options.WithBasicAuth("gituser", token))
	}
	
	client, err := nanogit.NewHTTPClient(repo, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("create nanogit client: %w", err)
	}

	return &NanoGitRepo{
		Repo:      repo,
		RepoToken: token,
		client:    client,
	}, nil
}

// isCommitHash checks if the revision looks like a git commit hash
func isNanoCommitHash(revision string) bool {
	// Match 7-40 character hex strings (typical git hash range)
	matched, _ := regexp.MatchString("^[a-f0-9]{7,40}$", revision)
	return matched
}

// Get retrieves a revision from a git repository into a target directory, optionally checkout specific directories
// This method provides the same interface as the go-git implementation but uses nanogit for better performance
// Returns the short revision hash that was retrieved
func (ng *NanoGitRepo) Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error) {
	// Set up logging context for nanogit if logger is available
	if ng.Lg != nil {
		ctx = log.ToContext(ctx, ng.Lg)
	}

	var commitHash hash.Hash
	var err error

	// Handle different revision types
	switch {
	case revision == "":
		// Get main/default branch
		ref, err := ng.client.GetRef(ctx, "refs/heads/main")
		if err != nil {
			// Fallback to master if main doesn't exist
			ref, err = ng.client.GetRef(ctx, "refs/heads/master")
			if err != nil {
				return "", fmt.Errorf("get default branch from %s: %w", ng.Repo, err)
			}
		}
		commitHash = ref.Hash

	case isNanoCommitHash(revision):
		// Handle both full and short commit hashes
		if len(revision) == 40 {
			// Full hash
			commitHash, err = hash.FromHex(revision)
			if err != nil {
				return "", fmt.Errorf("parse full commit hash %s: %w", revision, err)
			}
		} else {
			// Short hash - we need to resolve it to full hash
			// This is tricky with nanogit as it doesn't have direct short hash resolution
			// For now, return an error suggesting to use full hash
			return "", fmt.Errorf("nanogit requires full 40-character commit hashes, got %d characters: %s", len(revision), revision)
		}

	default:
		// Treat as branch/tag name
		ref, err := ng.client.GetRef(ctx, fmt.Sprintf("refs/heads/%s", revision))
		if err != nil {
			// Try as a tag
			ref, err = ng.client.GetRef(ctx, fmt.Sprintf("refs/tags/%s", revision))
			if err != nil {
				return "", fmt.Errorf("resolve reference %s from %s: %w", revision, ng.Repo, err)
			}
		}
		commitHash = ref.Hash
	}

	// Prepare clone options
	cloneOpts := nanogit.CloneOptions{
		Path: targetDir,
		Hash: commitHash,
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
	result, err := ng.client.Clone(ctx, cloneOpts)
	if err != nil {
		return "", fmt.Errorf("clone %s at %s to %s: %w", ng.Repo, revision, targetDir, err)
	}

	if ng.Lg != nil {
		ng.Lg.Info("Successfully cloned repository",
			"repo", ng.Repo,
			"revision", revision,
			"commit_hash", result.Commit.Hash.String()[:7],
			"target_dir", targetDir,
			"total_files", result.TotalFiles,
			"filtered_files", result.FilteredFiles)
	}

	// Return short commit hash (7 characters) to match go-git behavior
	return result.Commit.Hash.String()[:7], nil
}

// Additional helper methods that might be useful

// GetCommitHash resolves a revision to its full commit hash
func (ng *NanoGitRepo) GetCommitHash(ctx context.Context, revision string) (string, error) {
	if ng.Lg != nil {
		ctx = log.ToContext(ctx, ng.Lg)
	}

	switch {
	case revision == "":
		ref, err := ng.client.GetRef(ctx, "refs/heads/main")
		if err != nil {
			ref, err = ng.client.GetRef(ctx, "refs/heads/master")
			if err != nil {
				return "", fmt.Errorf("get default branch: %w", err)
			}
		}
		return ref.Hash.String(), nil

	case isNanoCommitHash(revision):
		return revision, nil

	default:
		ref, err := ng.client.GetRef(ctx, fmt.Sprintf("refs/heads/%s", revision))
		if err != nil {
			ref, err = ng.client.GetRef(ctx, fmt.Sprintf("refs/tags/%s", revision))
			if err != nil {
				return "", fmt.Errorf("resolve reference %s: %w", revision, err)
			}
		}
		return ref.Hash.String(), nil
	}
}