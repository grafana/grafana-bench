# Git Package

This package provides git repository access functionality with two implementations:
- **go-git**: Traditional Go-based git implementation 
- **nanogit**: High-performance git implementation optimized for speed

## Authentication

Both implementations support authentication for private repositories using tokens:

```go
// Public repository (no authentication)
repo, err := git.NewNanoGitSource("https://github.com/public/repo", "")

// Private repository with GitHub token
repo, err := git.NewNanoGitSource("https://github.com/private/repo", "ghp_your_token_here")
```

Both implementations use HTTP Basic Auth with the token as the password, which is the standard pattern for Git hosting providers.

## Running Tests

### Integration Tests

The nanogit integration tests require internet access to test against the remote test repository at `https://github.com/grafana/grafana-bench-git-test`.

**With internet access:**
```bash
go test ./pkg/git
```

**Without internet access:**
```bash
go test -short ./pkg/git
```

The `-short` flag skips integration tests that require network connectivity.