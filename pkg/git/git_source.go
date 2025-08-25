package git

import "context"

// GitSource defines the interface for getting the test suite from a git repository
type GitSource interface {
	// Get checkouts a test suite revision from a git source. Optionally, select directories to checkout.
	// Returns the short hash for the revision.
	// The revision can be a full reference (e.g. heads/branch-name), a short reference (e.g. v0.1.0)
	// or a commit hash.
	Get(ctx context.Context, targetDir string, revision string, checkoutDirs ...string) (string, error)
}