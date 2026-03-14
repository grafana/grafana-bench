# Architecture

This document covers how Grafana Bench is structured internally.

---

## Table of Contents

1. [Entry Point & CLI Structure](#1-entry-point--cli-structure)
2. [Configuration System](#2-configuration-system)
3. [Test Suite Lifecycle](#3-test-suite-lifecycle)
4. [Git Integration](#4-git-integration)
5. [Observability Integration](#5-observability-integration)
6. [File Locations](#6-file-locations)

---

## 1. Entry Point & CLI Structure

**Main Entry** (`bench.go`):
```
bench.go
  ↓
cmd/root (cobra command initialization)
  ↓
Subcommands:
  - test      (main test execution)
  - report    (post-execution reporting)
  - validate  (config/permission validation)
  - checkout  (git repository operations)
  - version   (version info)
```

**Exit Codes**:
- `0` - Success (all tests passed)
- `1` - Test failure (one or more tests failed)
- `2` - Internal error (configuration, execution, system error)

---

## 2. Configuration System

**Precedence Order** (highest to lowest):
1. **CLI Flags** - Explicit command-line arguments
2. **Environment Variables** - Uppercase with underscores (e.g., `SERVICE_URL`)
3. **Config File** - YAML file (default: `bench.yaml`)
4. **Defaults** - Hardcoded in flag definitions

**Implementation**:
- Uses **Viper** for configuration management
- Uses **Cobra** for CLI framework
- Flag loading in `pkg/utils/flags/flags.go`
- Configuration building in `pkg/config/config.go`

**Config File Format** (`bench.yaml`):
```yaml
test:
  runner: k6            # k6, playwright, go, gobench
  type: smoke           # smoke or load

suite:
  name: my-repo/smoke-tests
  path: ./tests

service:
  name: my-service
  url: http://localhost:3000
  version: 1.0.0

run:
  stage: ci

prometheus:
  metrics: true
  url: https://prometheus.example.com/api/v1/write

slack:
  notifications: true
  codeowners_mapping: ./codeowners-mapping.yaml
```

---

## 3. Test Suite Lifecycle

**Phase 1: Compilation & Fetching**
```
User invokes: bench test --suite-repo-url <url>
  ↓
TestCompiler (pkg/compile/compiler.go)
  ↓
Git Driver Selection:
  - nanogit (default, HTTP-only, fast)
  - gogit (supports SSH, full Git features)
  ↓
Clone/Checkout:
  - Supports specific commits, branches, tags
  - Sparse checkout via --suite-repo-dirs
  - Returns short commit hash (7 chars)
```

**Phase 2: Executor Selection**
```
TestExecutor Factory (config.BuildTestExecutor)
  ↓
Based on --test-runner:
  - "k6" → K6TestExecutor
  - "playwright" → PlaywrightTestExecutor
  - "go" → GoTestExecutor
  - "gobench" → GoBenchExecutor
```

**Phase 3: Test Execution**

Each executor follows this pattern:
1. **Setup**: Prepare environment, install dependencies
2. **Execution**: Run tests with framework-specific commands
3. **Parsing**: Parse framework output to common format
4. **Normalization**: Convert to `SuiteRunSummary` struct

**K6 Executor** (`pkg/executor/k6/`):
- Finds `.js` and `.ts` files
- Transpiles TypeScript with `k6pack`
- Runs each test with `k6 run --out json=<file>`
- Parses K6 JSON output
- Supports cloud output to Grafana Cloud K6

**Playwright Executor** (`pkg/executor/playwright/`):
- Runs prepare commands (e.g., `yarn install; playwright install`)
- Executes tests with `--reporter=json`
- Parses Playwright JSON report
- Extracts screenshots and traces

**Go Executor** (`pkg/executor/gotest/`):
- Runs `go test -json` for specified packages
- Streams JSON test events
- Implements **flaky test detection**: retries failed tests
- Tests passing after retry marked as "flaky"

**Phase 4: Result Normalization**

All executors produce `executor.SuiteRunSummary`:
```go
type SuiteRunSummary struct {
  SuiteName         string
  SuiteRevision     string
  StartTime         time.Time
  Status            SuiteStatus  // "passed" or "failed"
  TestsExecuted     int32
  TestsFailed       int32
  TestsFlaky        int32
  TestsPassed       int32
  TestsError        int32
  TotalDuration     time.Duration
  TestRuns          []TestRunSummary  // Per-test details
  Metrics           []metrics.Metric  // Custom metrics
}
```

**Phase 5: Reporting**

Reporter chain pattern (composite):
```
ChainReporter
  ↓
Executes in sequence:
  1. TextReporter / LogReporter (stdout)
  2. PrometheusReporter (if --prometheus-metrics)
  3. NotificationReporter (if --slack-notifications)
```

**Reporters**:
- **TextReporter**: Human-readable tabular output
- **LogReporter**: Structured JSON logs (slog)
- **PrometheusReporter**: Pushes metrics to Prometheus remote write
- **NotificationReporter**: Sends Slack messages via CODEOWNERS mapping

---

## 4. Git Integration

**Git Driver Abstraction** (`pkg/git/git.go`):
```go
type GitSource interface {
  Get(ctx, targetDir, revision, checkoutDirs...) (string, error)
}
```

**GoGit Driver** (`pkg/git/gogit/`):
- Pure Go implementation
- Supports SSH and HTTP
- Full Git feature set
- Slower for large repos

**Nanogit Driver** (`pkg/git/nanogit/`):
- Lightweight HTTP-only client
- Parallel object fetching
- Efficient for monorepos
- Default driver (faster)

**Sparse Checkout**:
```sh
--suite-repo-dirs CI/k6 --suite-repo-dirs CI/playwright
```
Only checks out specified directories.

---

## 5. Observability Integration

**Metrics** (`pkg/metrics/metric.go`):
- Parses Prometheus text exposition format
- Validates metric names (snake_case, base units)
- Sources: executor output, CLI flags, metrics files

**Default Metrics**:
```
bench_tests_executed
bench_tests_passed
bench_tests_failed
bench_tests_error
bench_tests_flaky
bench_total_duration_seconds
```

**Custom Metrics**:
- CLI: `--run-metric name{label=value}=42.5`
- File: `--run-metrics-file metrics.txt`
- Programmatic: Executors can emit metrics

**Prometheus Remote Write** (`pkg/prometheus/client.go`):
- Protocol Buffers format
- HTTP Basic Auth
- Configurable timeout
- Batched time series submission

**Slack Notifications** (`pkg/notifier/slack_notifier.go`):
- Reads CODEOWNERS file
- Maps file paths to team owners
- Looks up Slack channels via codeowners-mapping.yaml
- Sends formatted blocks with dashboard links

**Dashboard URL Templates** (`pkg/dashboard/dashboard.go`):
```
Template: http://grafana.example.com/d/dashboard?run={{.Id}}
Rendered: http://grafana.example.com/d/dashboard?run=abc123
```

---

## 6. File Locations

```
bench.go                          # Entry point
cmd/
  root/root.go                    # Root command
  test/command.go                 # Test execution
  report/report.go                # Reporting
  validate/validate.go            # Validation
  version/version.go              # Version info
  checkout/command.go             # Git checkout

pkg/
  config/config.go                # Configuration system
  executor/
    executor.go                   # Executor interface
    k6/                           # K6 executor + parser
    playwright/                   # Playwright executor + parser
    gotest/                       # Go executor + parser
    gobench/                      # Go benchmark executor + parser
  git/
    git.go                        # Git interface
    gogit/                        # GoGit implementation
    nanogit/                      # Nanogit implementation
  reporter/
    chain_reporter.go             # Composite reporter
    text_reporter.go              # Human-readable output
    log_reporter.go               # Structured logs
    prometheus_reporter.go        # Metrics push
    notification_reporter.go      # Slack notifications
  notifier/
    slack_notifier.go             # Slack integration
    codeowners_mapping.go         # CODEOWNERS parser
  prometheus/
    client.go                     # Remote write client
  metrics/
    metric.go                     # Metric parsing
    lint.go                       # Metric validation
  dashboard/
    dashboard.go                  # Dashboard URL rendering
  grafana/
    instance.go                   # Grafana client
  utils/
    logger/                       # Structured logging
    flags/flags.go                # Configuration loading

CI/                               # Test suites
  k6/                             # K6 test files
  playwright/                     # Playwright test files

docs/                             # Documentation
generators/
  doc/                            # Documentation generator
  libsonnet/                      # Libsonnet generator
libsonnet/                        # Generated libsonnet libraries
  versions.libsonnet
  experimental/main.libsonnet
```
