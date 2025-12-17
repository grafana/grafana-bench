# Running bench in Github Action CI

## Using the Setup Action (Recommended)

The easiest way to use grafana-bench in your GitHub Actions workflow is with the `setup-grafana-bench` action. This downloads and installs the pre-built binary for your platform.

### Basic Usage

```yaml
name: Test with Grafana Bench

on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]

jobs:
  bench-test:
    runs-on: ubuntu-latest
    services:
      grafana:
        image: grafana/grafana:latest
        ports:
          - 3000:3000
    steps:
      - uses: actions/checkout@v5
      
      - name: Setup Grafana Bench
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@339bd4f54dadc16432b1485dc3db733e34c72d27
        with:
          version: 'v0.6.6'
      
      - name: Run K6 API Tests
        run: |
          grafana-bench test \
            --test-type smoke \
            --grafana-url http://localhost:3000 \
            --test-suite CI/k6 \
            --log-level info
      
      - name: Run Playwright Tests
        run: |
          grafana-bench test \
            --test-runner playwright \
            --test-suite-base ./CI/playwright \
            --grafana-url http://localhost:3000 \
            --pw-prepare-cmd "npm install; npx playwright install" \
            --pw-execute-cmd "npm run test"
```

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | Version to install (e.g., `v0.6.6`) | Yes | N/A |

### Using from External Repositories

Since grafana-bench is a private repository, there are specific requirements for external repositories to use this action:

#### Prerequisites

1. **Organization membership**: Your repository must be part of the `grafana` GitHub organization
2. **Repository permissions**: The repository must have access to read from `grafana/grafana-bench`
3. **Token permissions**: Ensure your workflow has appropriate `contents: read` permissions

#### Setup

Add the `contents: read` permission to your workflow:

```yaml
name: Test with Grafana Bench

on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]

permissions:
  contents: read  # Required for accessing private repositories

jobs:
  bench-test:
    runs-on: ubuntu-latest
    # ... rest of your workflow
```

#### Usage

Reference the action using a specific commit hash (required for private repositories):

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@339bd4f54dadc16432b1485dc3db733e34c72d27
  with:
    version: 'v0.6.6'
```

The action automatically uses the workflow's `GITHUB_TOKEN` to authenticate with the GitHub API for downloading release binaries from the private repository. The token is passed implicitly via `${{ github.token }}` - no additional configuration is required for repositories within the Grafana organization.

**Note**: If you encounter authentication issues, ensure your repository has the "Allow GitHub Actions to access other repositories in your organization" setting enabled in the repository settings under Actions > General.

#### Updating the Commit Reference

The commit SHA references in this documentation are automatically updated when you run `make docs`. This ensures the examples always reference the latest version of the action.

When the action is updated, run `make docs` to update all documentation references automatically. The CI workflow will also create a PR with updated documentation if changes are detected.

### Platform Support

The action automatically detects your platform and installs the appropriate binary:

- **Linux**: amd64, arm64
- **macOS**: amd64, arm64  
- **Windows**: amd64

The action uses GitHub's API with authentication to download binaries from private releases. If binary download fails, it falls back to `go install` with proper authentication for private repositories.

### Exit Codes for CI Integration

Grafana Bench uses distinct exit codes to help CI systems differentiate between test failures and internal errors:

- **Exit code 0**: Success - all tests passed
- **Exit code 1**: Test failure - one or more tests failed
- **Exit code 2**: Internal error - configuration, execution, or system error

This allows your CI workflows to handle test failures differently from internal errors. For example, test failures might trigger notifications to developers, while internal errors might page the infrastructure team.

### When to Use Setup Action vs Docker

**Use the Setup Action when:**

- Running Go tests (`go test`)
- Using bench for reporting only
- You want to manage dependencies yourself (K6, Playwright, etc.)
- Faster startup time is preferred

**Use Docker when:**

- You need pre-installed dependencies (K6, Playwright, browsers)
- You want a consistent, isolated environment
- Your tests require specific system dependencies

**Setup Action Example (for Go tests/reporting):**

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@339bd4f54dadc16432b1485dc3db733e34c72d27
  with:
    version: 'v0.6.6'

- name: Run Go tests with bench reporter
  run: |
    grafana-bench test \
      --test-runner gotest \
      --test-suite-base ./tests \
      --grafana-url http://localhost:3000
```

**Docker Example (with pre-installed dependencies):**

```yaml
- name: Run Playwright tests with Docker
  run: |
    docker run --rm \
      --network=host \
      --volume="./:/tests/" \
      us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.6 test \
      --test-runner playwright \
      --grafana-url "http://localhost:3000"
```

## Docker-based CI Example

This is an abreviated version of the [CI used for Bench](../.github/workflows/ci.yaml).

```yaml
name: Bench CI

on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]
    
jobs:
  bench-test:
    runs-on: ubuntu-latest

    # run grafana instance
    services:
      grafana:
        image: grafana/grafana:latest
        ports:
    # bench test
    steps:
      - name: checkout code
        uses: actions/checkout@v4
      - name: check playwright test
        run: |
          docker run --rm \
            --network=host \
            --volume="./CI/:/tests/CI/" \

            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.6 test \
            --grafana-url "http://localhost:3000" \
            --grafana-admin-user "admin" \
            --grafana-admin-password "admin" \
            --test-suite-base "/tests/CI/plugin-e2e"  \
            --test-runner "playwright" \
            --pw-prepare-cmd "yarn install; playwright install chromium" \
            --pw-execute-cmd "yarn run test" \
            --log-level DEBUG
      - name: archive screenshots
        uses: actions/upload-artifact@v3
        with:
          name: screenshots
          path: screenshots
```

## Datasource workflow example

This is a live example of a workflow from the [clickhouse datasource](https://github.com/grafana/clickhouse-datasource/blob/main/.github/workflows/grafana-bench.yml).

```yaml
name: Grafana Bench
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test:
    timeout-minutes: 60
    runs-on: ubuntu-latest
    steps:
    # Plugin setup
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'yarn'

      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      - name: Build backend
        uses: magefile/mage-action@v3
        with:
          args: buildAll
          version: latest

      - name: Install frontend dependencies
        run: yarn install --frozen-lockfile

      - name: Build frontend
        run: yarn build
        env:
          NODE_OPTIONS: '--max_old_space_size=4096'

      - name: Install and run Docker Compose
        uses: hoverkraft-tech/compose-action@v2.0.2
        with:
          compose-file: './docker-compose.yml'
    # Bench
      - name: Ensure Grafana is running
        run: |
          curl http://localhost:3000

      - name: Run Grafana Bench tests
        run: |
          docker run --rm \
            --network=host \
            --volume="./:/tests/" \
            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.6 test \
            --test-runner "playwright" \
            --test-suite-base "/tests/" \
            --grafana-url "http://localhost:3000" \
            --pw-prepare-cmd "yarn install --frozen-lockfile; playwright install chromium" \
            --pw-execute-cmd "yarn e2e" \
            --test-env "CI=true" \
            --log-level DEBUG 
```

### Workflow Breakdown

We configure the tests to run on every PR and merge to Main. This gives us early and often feedback.

```yaml
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]
```

This workflow first configures the plugin:

1. checkout the code
2. install node
3. install go
4. build the backend
5. install frontend dependencies
6. install docker-compose to run the plugin

Then we run Bench:

7. ensure the container is running
8. invoke bench against the container

## Exporting logs to centralized loki database

In development
