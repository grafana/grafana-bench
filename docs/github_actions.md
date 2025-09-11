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
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@v1
        with:
          version: 'v0.6.1'
      
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
| `version` | Version to install (e.g., `v0.6.1`) | Yes | N/A |

### Platform Support

The action automatically detects your platform and installs the appropriate binary:
- **Linux**: amd64, arm64
- **macOS**: amd64, arm64  
- **Windows**: amd64

If binary download fails, it falls back to `go install github.com/grafana/grafana-bench@<version>`.

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
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@v1
  with:
    version: 'v0.6.1'

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
      --volume="./:/home/bench/tests/" \
      us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.1 test \
      --test-runner playwright \
      --test-suite-base "/home/bench/tests/" \
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
            --volume="./CI/:/home/bench/tests/CI/" \

            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.1 test \
            --grafana-url "http://localhost:3000" \
            --grafana-admin-user "admin" \
            --grafana-admin-password "admin" \
            --test-suite-base "/home/bench/tests/CI/plugin-e2e"  \
            --test-runner "playwright" \
            --pw-prepare-cmd "yarn install; yarn playwright install" \
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
            --volume="./:/home/bench/tests/" \
            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.1 test \
            --test-runner "playwright" \
            --test-suite-base "/home/bench/tests/" \
            --grafana-url "http://localhost:3000" \
            --pw-prepare-cmd "yarn install --frozen-lockfile; yarn playwright install" \
            --pw-execute-cmd "yarn e2e" \
            --test-env-vars "CI=true" \
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