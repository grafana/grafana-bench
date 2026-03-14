# Running bench in GitHub Actions

## Installation

There are two ways to use bench in CI: the **setup action** (installs the binary) or **Docker** (pre-bundled with K6/Playwright dependencies).

### Setup Action

The `setup-grafana-bench` action downloads and installs the pre-built binary for your platform.

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.3'
```

**Supported platforms:** Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64). If binary download fails, the action falls back to `go install`.

| Input | Description | Required |
|-------|-------------|----------|
| `version` | Version to install (e.g. `v1.0.3`) | Yes |

### When to use Docker instead

**Use the setup action when:**
- Running Go tests or benchmarks
- Using bench for reporting only (`bench report`)
- You want to manage test dependencies yourself

**Use Docker when:**
- You need pre-installed K6 or Playwright + browsers
- You want a consistent, isolated environment

---

## Basic examples

### K6 tests (setup action)

```yaml
jobs:
  bench-test:
    runs-on: ubuntu-latest
    services:
      grafana:
        image: grafana/grafana:latest
        ports:
          - 3000:3000
    steps:
      - uses: actions/checkout@v4

      - name: Setup Grafana Bench
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
        with:
          version: 'v1.0.3'

      - name: Run K6 tests
        run: |
          grafana-bench test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --test-type smoke \
            --suite-path CI/k6 \
            --suite-name my-repo/ci/k6 \
            --run-stage ci \
            --report-output log
```

### Playwright tests (Docker)

```yaml
      - name: Run Playwright tests
        run: |
          docker run --rm \
            --network=host \
            --volume="./:/tests/" \
            ghcr.io/grafana/grafana-bench-playwright:v1.0.3 test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --test-runner playwright \
            --test-type smoke \
            --suite-path /tests/CI/playwright \
            --suite-name my-repo/ci/playwright \
            --run-stage ci \
            --report-output log \
            --pw-prepare "yarn install; playwright install chromium" \
            --pw-execute "yarn run test"
```

> For Playwright troubleshooting (including common permission errors), see the [Playwright guide](writing_pw_tests.md#troubleshooting).

### Go tests (setup action)

```yaml
      - name: Run Go tests
        run: |
          grafana-bench test \
            --service my-service \
            --service-version latest \
            --test-runner gotest \
            --test-type smoke \
            --suite-path ./tests \
            --suite-name my-repo/tests \
            --run-stage ci \
            --report-output log
```

---

## Suite naming

The `--suite-name` flag is **required** and identifies your tests in logs and Prometheus metrics.

**Recommended format:** `<project>/<test-type>`

- `my-plugin/e2e-tests`
- `api-service/smoke-tests`
- `grafana/k6-load-tests`

Suite names become Prometheus metric labels (`suite_name="my-plugin/e2e-tests"`), so use a consistent convention you can filter on in dashboards.

---

## With Prometheus metrics

Add `--prometheus-metrics` and provide credentials via GitHub Actions secrets:

```yaml
      - name: Run tests with metrics
        env:
          PROMETHEUS_URL: ${{ secrets.PROMETHEUS_URL }}
          PROMETHEUS_USER: ${{ secrets.PROMETHEUS_USER }}
          PROMETHEUS_PASSWORD: ${{ secrets.PROMETHEUS_PASSWORD }}
        run: |
          grafana-bench test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --suite-path CI/k6 \
            --suite-name my-repo/ci/k6 \
            --run-stage ci \
            --report-output log \
            --prometheus-metrics
```

When using Docker, pass the variables explicitly with `-e`:

```yaml
        run: |
          docker run --rm \
            --network=host \
            -e PROMETHEUS_URL \
            -e PROMETHEUS_USER \
            -e PROMETHEUS_PASSWORD \
            ghcr.io/grafana/grafana-bench:v1.0.3 test \
            --service grafana \
            --service-url http://localhost:3000 \
            --service-version latest \
            --suite-path CI/k6 \
            --suite-name my-repo/ci/k6 \
            --run-stage ci \
            --report-output log \
            --prometheus-metrics
```

See the [metrics guide](metrics.md) for full configuration details.

---

## Exit codes

Bench uses distinct exit codes so CI can differentiate test failures from internal errors:

| Code | Meaning |
|------|---------|
| `0` | Success — all tests passed |
| `1` | Test failure — one or more tests failed |
| `2` | Internal error — configuration, execution, or system error |

---

## Real-world example: datasource plugin

From the [clickhouse datasource](https://github.com/grafana/clickhouse-datasource/blob/main/.github/workflows/grafana-bench.yml):

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

      - name: Ensure Grafana is running
        run: curl http://localhost:3000

      - name: Run Grafana Bench tests
        run: |
          docker run --rm \
            --network=host \
            --volume="./:/tests/" \
            ghcr.io/grafana/grafana-bench:v1.0.3 test \
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
            --test-env "CI=true"
```

This workflow runs on every PR and merge to main. It builds the plugin first, starts the stack with Docker Compose, then runs bench against it.

---

## Related pages

- [Slack notifications guide](notifications.md) - Route failure alerts to team Slack channels
- [Metrics guide](metrics.md) - Export test results to Prometheus
- [bench test reference](bench_test.md) - Full flag reference
