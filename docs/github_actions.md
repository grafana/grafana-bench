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
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
        with:
          version: 'v1.0.1'

      - name: Run K6 API Tests
        run: |
          grafana-bench test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --test-type smoke \
            --suite-path CI/k6 \
            --suite-name my-repo/ci/k6 \
            --run-stage ci \
            --report-output log \
            --log-level info

      - name: Run Playwright Tests
        run: |
          grafana-bench test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --test-runner playwright \
            --test-type smoke \
            --suite-path ./CI/playwright \
            --suite-name my-repo/ci/playwright \
            --run-stage ci \
            --report-output log \
            --pw-prepare "npm install; npx playwright install" \
            --pw-execute "npm run test"
```

> **Note:** For Playwright troubleshooting (including common permission errors), see the [Playwright Troubleshooting Guide](writing_pw_tests.md#troubleshooting).

### Suite Naming Best Practices

The `--suite-name` flag is **required** and identifies your tests in logs and Prometheus metrics. Use a consistent naming convention:

**Recommended Format:** `<project>/<test-type>`

**Examples:**
- `grafana-bench/go-tests` - Go unit/integration tests
- `grafana-bench/benchmarks` - Performance benchmarks
- `my-plugin/e2e-tests` - End-to-end browser tests
- `api-service/smoke-tests` - API smoke tests
- `grafana/k6-load-tests` - Load testing suite

**Why This Matters:**
- Suite names become Prometheus metric labels (`suite_name="grafana-bench/go-tests"`)
- Enables filtering and grouping in dashboards
- Makes it easy to track metrics over time per test suite
- Helps identify which tests are failing in aggregate views

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `version` | Version to install (e.g., `v1.0.1`) | Yes | N/A |

### Authentication and CI Tokens

The `setup-grafana-bench` action automatically fetches shared CI tokens from Vault when it runs. These tokens enable optional features like Prometheus metrics reporting. This works whether you're using the bench binary directly or via Docker.

#### Using Tokens with Direct Binary

After the setup step completes, the tokens are available in your environment. Simply add the `--prometheus-metrics` flag to your bench commands:

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.1'

- name: Run tests with Prometheus metrics
  run: |
    grafana-bench test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version latest \
      --test-runner playwright \
      --test-type smoke \
      --suite-path ./CI/playwright \
      --suite-name my-repo/ci/playwright \
      --run-stage ci \
      --report-output log \
      --prometheus-metrics
```

#### Using Tokens with Docker

When using Docker, you need to explicitly pass the Prometheus environment variables to the container using `-e` flags:

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.1'

- name: Run tests in Docker with Prometheus metrics
  run: |
    docker run --rm \
      --network=host \
      --volume="./:/tests/" \
      -e PROMETHEUS_URL="${PROMETHEUS_URL}" \
      -e PROMETHEUS_USER="${PROMETHEUS_USER}" \
      -e PROMETHEUS_PASSWORD="${PROMETHEUS_PASSWORD}" \
      us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v1.0.1 test \
      --service grafana \
      --service-url "http://localhost:3000" \
      --service-version latest \
      --test-runner playwright \
      --test-type smoke \
      --suite-path /tests/CI/playwright \
      --suite-name my-repo/ci/playwright \
      --run-stage ci \
      --report-output log \
      --prometheus-metrics
```

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
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.1'
```

The action automatically uses the workflow's `GITHUB_TOKEN` to authenticate with the GitHub API for downloading release binaries from the private repository. The token is passed implicitly via `${{ github.token }}` - no additional configuration is required for repositories within the Grafana organization.

**Note**: If you encounter authentication issues, ensure your repository has the "Allow GitHub Actions to access other repositories in your organization" setting enabled in the repository settings under Actions > General.

#### Updating the Commit Reference

The commit SHA references in this documentation are automatically updated when you run `make docs`. The documentation always references the latest commit on the **main branch** that modified the action file, preventing circular updates during development.

When the action is updated and merged to main, run `make docs` to update all documentation references automatically. The CI workflow will also create a PR with updated documentation if changes are detected.

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
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.1'

- name: Run Go tests with bench reporter
  run: |
    grafana-bench test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version latest \
      --test-runner gotest \
      --test-type smoke \
      --suite-path ./tests \
      --suite-name my-repo/tests \
      --run-stage ci \
      --report-output log
```

**Docker Example (with pre-installed dependencies):**

```yaml
- name: Run Playwright tests with Docker
  run: |
    docker run --rm \
      --network=host \
      --volume="./:/tests/" \
      us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v1.0.1 test \
      --service grafana \
      --service-url "http://localhost:3000" \
      --service-version latest \
      --test-runner playwright \
      --test-type smoke \
      --suite-path /tests/CI/playwright \
      --suite-name my-repo/ci/playwright \
      --run-stage ci \
      --report-output log
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
            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v1.0.1 test \
            --service grafana \
            --service-url "http://localhost:3000" \
            --service-version latest \
            --test-runner "playwright" \
            --test-type smoke \
            --suite-path "/tests/CI/plugin-e2e" \
            --suite-name my-repo/ci/plugin-e2e \
            --run-stage ci \
            --report-output log \
            --pw-prepare "yarn install; playwright install chromium" \
            --pw-execute "yarn run test" \
            --test-env "GRAFANA_USER=admin" \
            --test-env "GRAFANA_PASSWORD=admin" \
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
            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v1.0.1 test \
            --service grafana \
            --service-url "http://localhost:3000" \
            --service-version latest \
            --test-runner "playwright" \
            --test-type smoke \
            --suite-path "/tests/" \
            --suite-name clickhouse-datasource/e2e \
            --run-stage ci \
            --report-output log \
            --pw-prepare "yarn install --frozen-lockfile; playwright install chromium" \
            --pw-execute "yarn e2e" \
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
