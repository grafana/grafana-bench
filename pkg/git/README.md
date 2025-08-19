# Git Package

This package provides git repository access functionality with two implementations:
- **go-git**: Traditional Go-based git implementation 
- **nanogit**: High-performance git implementation optimized for speed

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