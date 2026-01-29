# Setup Grafana Bench Action

This GitHub Action installs the grafana-bench binary and sets up Prometheus secrets for metrics export.

## Inputs

### `version`

**Required** - Version of grafana-bench to install.

- Use a specific version tag (e.g., `"v0.6.1"`)
- Use `"latest"` to install the most recent release
- Use `"false"` to skip installation and only setup Prometheus secrets

## Outputs

None. The action sets up the following environment variables:

- `PROMETHEUS_URL` - Prometheus push endpoint URL
- `PROMETHEUS_USER` - Prometheus username
- `PROMETHEUS_PASSWORD` - Prometheus authentication token

## Usage

### Install specific version with secrets

```yaml
- uses: ./.github/actions/setup-grafana-bench
  with:
    version: 'v0.6.11'

- name: Run benchmarks
  run: |
    grafana-bench test \
      --service myservice \
      --service-version v1.0.0 \
      --test-runner gobench \
      --suite-path ./benchmarks \
      --prometheus-metrics
```

### Install latest version with secrets

```yaml
- uses: ./.github/actions/setup-grafana-bench
  with:
    version: 'latest'
```

### Setup secrets only (skip installation)

Useful when you want to install bench from source or use a different installation method:

```yaml
- uses: actions/checkout@v5

- uses: actions/setup-go@v5
  with:
    go-version-file: go.mod

- name: Setup Prometheus secrets
  uses: ./.github/actions/setup-grafana-bench
  with:
    version: false  # Only setup secrets, skip installation

- name: Install from source
  run: go install

- name: Run benchmarks with metrics
  run: |
    grafana-bench test \
      --service grafana-bench \
      --service-version ${{ github.sha }} \
      --test-runner gobench \
      --suite-path ./pkg/executor/gobench/tests \
      --prometheus-metrics
```

## How It Works

1. **Installation** (if version != 'false'):
   - Sets up Go environment
   - Downloads prebuilt binary for the platform/architecture
   - Falls back to `go install` if binary is not available
   - Verifies installation with `grafana-bench version`

2. **Secrets Setup** (always runs):
   - Fetches Prometheus credentials from Vault
   - Sets environment variables for `grafana-bench` to use
   - These variables are automatically picked up by the `--prometheus-metrics` flag

## Requirements

- **Permissions**: Requires `id-token: write` and `contents: read` for Vault secrets access
- **Context**: Must run in a GitHub Actions workflow with access to Grafana's shared Vault secrets

## Examples in This Repository

See `.github/workflows/ci.yml` for real-world usage examples:

- `test-go` job: Installs specific version (v0.6.11) with secrets
- `test-benchmarks` job: Skips installation (version: false) and installs from source
- `test-setup-action` job: Tests both binary download and go install fallback
