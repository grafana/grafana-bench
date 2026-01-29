# Configuration Guide

This guide explains how to configure Grafana Bench using command-line flags, environment variables, and configuration files.

---

## Understanding Flag Organization

Bench flags are organized by **prefix** to help you quickly identify what each flag controls. When you're setting up a test run, think about these categories in order:

### Core Categories

1. **`--suite-*`** - Where are your tests?
2. **`--test-*`** - How should tests be executed?
3. **`--service-*`** - What service is being tested?
4. **`--run-*`** - Metadata about this test run

### Optional Categories (as needed)

5. **Test Runner Specific** - `--k6-*`, `--pw-*`, `--go-*`, `--gobench-*`
6. **Output & Notifications** - `--report-*`, `--slack-*`, `--prometheus-*`
7. **Advanced** - `--bench-*`, `--git-*`

---

## Flag Categories Reference

### `--suite-*` Test Suite Source

Defines **where your tests are located** and which version to use.

| Flag | Required | Description |
|------|----------|-------------|
| `--suite-name` | ✅ | Identifier for your test suite (e.g., `my-repo/smoke-tests`) |
| `--suite-path` | | Path to tests relative to `--suite-base` |
| `--suite-base` | | Base directory for test suite (default: current directory) |
| `--suite-repo-url` | | Git repository URL containing tests |
| `--suite-repo-token` | | Authentication token for private repositories |
| `--suite-repo-dirs` | | Specific directories to checkout from repo |
| `--suite-revision` | | Git revision/branch/tag to checkout |

**Example:**
```bash
# Local tests
grafana-bench test \
  --suite-name my-repo/smoke-tests \
  --suite-path ./tests

# Tests from Git repo
grafana-bench test \
  --suite-name my-repo/e2e-tests \
  --suite-repo-url https://github.com/org/tests.git \
  --suite-revision main \
  --suite-path e2e
```

---

### `--test-*` Test Execution

Controls **how tests are executed** (which runner, what type, environment variables).

| Flag | Description |
|------|-------------|
| `--test-runner` | Which executor to use: `k6`, `playwright`, `go`, `gobench` (default: `k6`) |
| `--test-type` | Test type: `smoke` or `load` (default: `smoke`) |
| `--test-env` | Environment variables for test execution (format: `KEY=value` or `KEY`) |
| `--test-verbose` | Show detailed test output |

**Example:**
```bash
grafana-bench test \
  --suite-name my-repo/tests \
  --test-runner go \
  --test-type smoke \
  --test-env "DEBUG=1" \
  --test-env "API_TOKEN"  # passthrough from environment
```

---

### `--service-*` Service Under Test

Identifies **what service is being tested** and how to connect to it.

| Flag | Required | Description |
|------|----------|-------------|
| `--service` | ✅ | Service name (e.g., `grafana`, `loki`, `tempo`) |
| `--service-version` | ✅* | Version being tested (e.g., `11.0.0`) |
| `--service-url` | | Service URL (default: `http://localhost:3000`) |
| `--service-timeout` | | Timeout for health checks (default: `1m`) |
| `--service-health-check` | | Perform TCP health check before tests |
| `--fetch-grafana-version` | ✅* | Alternative to `--service-version`: fetch from Grafana API (`user:pass`) |

*Either `--service-version` or `--fetch-grafana-version` is required.

**Example:**
```bash
# Explicit version
grafana-bench test \
  --service grafana \
  --service-version 11.0.0 \
  --service-url http://localhost:3000

# Fetch version from API
grafana-bench test \
  --service grafana \
  --fetch-grafana-version admin:admin \
  --service-url http://localhost:3000
```

---

### `--run-*` Suite Run Metadata

Adds **contextual metadata** about this specific test execution (environment, custom metrics, attributes).

| Flag | Description |
|------|-------------|
| `--run-stage` | Execution stage/environment (e.g., `local`, `ci`, `rrc`, `prod`) (default: `local`) |
| `--run-dashboard` | Dashboard URL template with `{{.Id}}` substitution |
| `--run-attribute` | Custom key-value attributes (format: `key=value`) |
| `--run-metric` | Custom metrics (format: `name{label=value}=123.45`) |
| `--run-metrics-file` | File with Prometheus-format metrics |
| `--run-metrics-prefix` | Prefix for custom metric names |

**Use cases:**
- `--run-stage`: Distinguish between `local` dev runs, `ci` builds, `rrc` releases, and `prod` monitoring
- `--run-attribute`: Add deployment metadata (e.g., `channel=grafana-rrc-alerts`, `region=us-east-1`)
- `--run-metric`: Track custom performance metrics alongside test results

**Example:**
```bash
grafana-bench test \
  --suite-name my-repo/tests \
  --run-stage rrc \
  --run-attribute "channel=grafana-rrc" \
  --run-attribute "release=v11.0.0-rc1" \
  --run-metric "deploy_duration_seconds=45.2"
```

---

## Test Runner Specific Flags

### `--k6-*` K6 Executor

For K6 JavaScript API tests.

| Flag | Description |
|------|-------------|
| `--k6-cloud-output` | Send results to Grafana Cloud K6 |
| `--k6-cloud-token` | GCK6 access token |
| `--k6-cloud-project` | GCK6 project ID |

### `--pw-*` Playwright Executor

For Playwright browser tests.

| Flag | Description |
|------|-------------|
| `--pw-prepare` | Setup commands (e.g., `npm install; playwright install chromium`) |
| `--pw-execute` | Test execution command (e.g., `npm test`) |

### `--go-*` Go Test Executor

For Go unit/integration tests.

| Flag | Description |
|------|-------------|
| `--go-test-packages` | Package patterns (e.g., `./...`, `./pkg/...`) |
| `--go-args` | Arguments to `go test` (e.g., `-tags=integration -race`) |
| `--go-test-args` | Arguments passed to tests via `-args` |
| `--go-retries` | Retry count for failed tests (reports flakiness) |

### `--gobench-*` Go Benchmark Executor

For Go performance benchmarks.

| Flag | Description |
|------|-------------|
| `--gobench-packages` | Package patterns (default: `./...`) |
| `--gobench-pattern` | Benchmark name regex (default: `.` = all) |
| `--gobench-time` | Duration per benchmark (e.g., `10s`, `100x`) |
| `--gobench-count` | Times to run each benchmark (default: 1) |
| `--gobench-mem` | Enable memory stats (default: `true`) |
| `--gobench-args` | Arguments to `go test` |
| `--gobench-bench-args` | Arguments to benchmarks via `-args` |

---

## Output & Notifications

### `--report-*` Report Output

| Flag | Description |
|------|-------------|
| `--report-output` | Format: `text` (human-readable), `log` (structured), `json` (default: `text`) |

### `--slack-*` Slack Notifications

| Flag | Description |
|------|-------------|
| `--slack-notifications` | Enable Slack notifications for failures |
| `--slack-passing` | Also notify on passing test suites |
| `--slack-token` | Slack bot token (requires `chat:write`, `channels:read`) |
| `--slack-codeowners-mapping` | Path to CODEOWNERS → Slack channel mapping file |

### `--prometheus-*` Prometheus Metrics

| Flag | Description |
|------|-------------|
| `--prometheus-metrics` | Send metrics to Prometheus remote write endpoint |
| `--prometheus-url` | Remote write URL |
| `--prometheus-user` | Basic auth username |
| `--prometheus-password` | Basic auth password |
| `--prometheus-timeout` | Request timeout |
| `--prometheus-strict-lint` | Fail on invalid metrics |

---

## Configuration File

Instead of long command lines, use a `bench.yaml` configuration file:

```yaml
# bench.yaml
test:
  runner: k6
  type: smoke
  env:
    VAR1: "value1"
    VAR2: "value2"

suite:
  name: "my-repo/smoke-tests"
  path: "./tests"

service:
  name: grafana
  url: "http://localhost:3000"
  version: "11.0.0"

run:
  stage: ci

slack:
  notifications: true
  token: "xoxb-your-token"

prometheus:
  metrics: true
  url: "https://prometheus.example.com/api/v1/write"
```

**Load the file:**
```bash
grafana-bench test --config bench.yaml
```

**Override from command line:**
```bash
grafana-bench test --config bench.yaml --run-stage prod
```

**Precedence:** CLI flags > Environment variables > Config file

---

## Environment Variables

Most flags have corresponding environment variables:

| Flag | Environment Variable |
|------|---------------------|
| `--suite-name` | `SUITE_NAME` |
| `--suite-repo-url` | `SUITE_REPO_URL` |
| `--suite-repo-token` | `SUITE_REPO_TOKEN` |
| `--suite-revision` | `SUITE_REVISION` |
| `--service-url` | `SERVICE_URL` |
| `--service-version` | `SERVICE_VERSION` |
| `--fetch-grafana-version` | `FETCH_GRAFANA_VERSION` |
| `--slack-token` | `SLACK_TOKEN` |
| `--prometheus-url` | `PROMETHEUS_URL` |
| `--prometheus-user` | `PROMETHEUS_USER` |
| `--prometheus-password` | `PROMETHEUS_PASSWORD` |
| `--k6-cloud-token` | `K6_CLOUD_TOKEN` |
| `--k6-cloud-project` | `K6_CLOUD_PROJECT_ID` |
| `--bench-revision` | `BENCH_REVISION` |

**Use a `.env` file for local development:**
```bash
# .env
SLACK_TOKEN=xoxb-your-token
PROMETHEUS_URL=https://prometheus.example.com/api/v1/write
PROMETHEUS_USER=admin
PROMETHEUS_PASSWORD=secret
```

Bench automatically loads `.env` from your working directory.

---

## Quick Start Examples

### Minimal Local Test
```bash
grafana-bench test \
  --suite-name my-repo/tests \
  --suite-path ./tests \
  --service grafana \
  --service-version 11.0.0
```

### CI Integration
```bash
grafana-bench test \
  --suite-name my-repo/e2e-tests \
  --suite-path ./e2e \
  --service grafana \
  --service-url http://grafana:3000 \
  --service-version "${GRAFANA_VERSION}" \
  --run-stage ci \
  --prometheus-metrics \
  --slack-notifications
```

### Go Benchmarks with Metrics
```bash
grafana-bench test \
  --suite-name my-repo/benchmarks \
  --service bench \
  --service-version "${GIT_SHA}" \
  --test-runner gobench \
  --gobench-pattern "BenchmarkAPI" \
  --gobench-time 10s \
  --run-stage ci \
  --prometheus-metrics
```

---

## Next Steps

- **[Writing K6 Tests](writing_k6_api_tests.md)** - API testing guide
- **[Writing Playwright Tests](writing_pw_tests.md)** - Browser testing guide
- **[Writing Go Tests](writing_go_tests.md)** - Go testing guide
- **[Metrics Guide](metrics.md)** - Prometheus metrics deep dive
- **[Notifications Guide](notifications.md)** - Slack integration setup
- **[GitHub Actions Integration](github_actions.md)** - CI/CD setup
