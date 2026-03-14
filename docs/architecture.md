# Grafana Bench - Architecture & Integration Guide

**Version:** v1.0.3
**Last Updated:** 2026-01-21

This document provides comprehensive architectural knowledge about Grafana Bench, its internal implementation, deployment integration, and observability setup. This is the definitive reference for understanding how bench works end-to-end.

---

## Table of Contents

1. [Overview](#overview)
2. [Internal Architecture](#internal-architecture)
3. [Deployment Integration](#deployment-integration)
4. [Observability & Monitoring](#observability--monitoring)
5. [Key Integration Points](#key-integration-points)
6. [File Locations](#file-locations)

---

## Overview

### What is Bench?

Grafana Bench is a CLI tool that provides:
- **Unified test execution** across K6, Playwright, and Go test frameworks
- **Standardized structured logging** for test observability
- **Metrics collection and reporting** to Prometheus
- **Slack notifications** with CODEOWNERS-based routing
- **Git-based test suite management** with sparse checkout support
- **Multi-environment support** (local, CI, release pipelines)

### Core Value Proposition

1. **Portability**: Same tests run locally, in CI, and in production deployments
2. **Observability**: Structured logs and metrics enable tracking test health over time
3. **Team Autonomy**: Self-service test authoring with automatic team routing
4. **Standardization**: Common interface across different test frameworks

---

## Internal Architecture

### 1. Entry Point & CLI Structure

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

### 2. Configuration System

**Precedence Order** (highest to lowest):
1. **CLI Flags** - Explicit command-line arguments
2. **Environment Variables** - Uppercase with underscores (e.g., `GRAFANA_URL`)
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

### 3. Test Suite Lifecycle

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

### 4. Git Integration

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
```bash
--suite-repo-dirs CI/k6 --suite-repo-dirs CI/playwright
```
Only checks out specified directories.

### 5. Observability Integration

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

---

## Deployment Integration

### 1. Bench in deployment_tools

**Location**: `/Users/jeff/projects/deployment_tools/ksonnet/`

**Integration Layers**:
```
CLI (grafana-bench binary)
  ↓
libsonnet/bench/v1.0.3/main.libsonnet (config builder)
  ↓
argo-workflows-util/bench-v2.libsonnet (Argo integration)
  ↓
Argo Workflow Templates (bench_test template)
  ↓
Deployment Workflows (hosted-grafana-cd, unified-storage-cd, etc.)
```

### 2. Bench Libsonnet Code

**File**: `ksonnet/lib/bench/v1.0.3/main.libsonnet`

**Auto-generated from CLI**:
- ~50+ configuration options
- Maps CLI flags to libsonnet parameters
- Builds complete bench command

**Example Usage**:
```jsonnet
local bench = import 'ksonnet/lib/bench/v1.0.3/main.libsonnet';

bench.suite({
  suitePath: 'CI/k6',
  testType: 'smoke',
  testRunner: 'k6',
  grafanaUrl: 'http://grafana:3000',
  prometheusMetrics: true,
  slackNotifications: true,
})
```

**Version Management** (`ksonnet/lib/bench/versions.libsonnet`):
- Registry of available versions
- Dynamic import based on version string
- Available: `experimental`, `v1.0.3`

**Image Selection**:
```
Test Runner → Image
─────────────────────
k6          → grafana-bench:v1.0.3
playwright  → grafana-bench-playwright:v1.0.3
go          → grafana-bench:v1.0.3
```

### 3. Argo Workflow Integration

**Modern Interface** (`bench-v2.libsonnet`):
```jsonnet
local aw = import 'argo-workflows-util/index.libsonnet';

aw.testingSteps.benchV2('test-step', 'v1.0.3')
  .withBenchTest('http://grafana:3000', {
    suitePath: 'CI/k6',
    testType: 'smoke',
  })
```

**Features**:
- Workflow-level version management
- Automatic image selection
- Simplified API

**Legacy Interface** (`bench.libsonnet`):
- Suite-level version via `benchRevision`
- Still supported for backward compatibility

### 4. Grafana Bench CD (Self-Testing)

**Location**: `ksonnet/environments/grafana-bench-cd/`

**Purpose**: Continuous integration for bench itself

**Architecture**:
- **Embedded Grafana**: Each test gets fresh Grafana container sidecar
- **Parallel Execution**: All tests run in parallel
- **Health Checks**: Waits for Grafana readiness (2min startup + 5min probe)
- **Test Coverage**: K6 and Playwright runners validated

**Test Suites**:
1. `bench-k6-smoke` - Tests K6 API testing
2. `bench-playwright-smoke` - Tests Playwright browser testing

**Notifications**:
- Slack: `#grafana-bench-ops`
- Oncall: `@grafana-developer-enablement-squad`

### 5. Hosted Grafana CD Integration

**Location**: `ksonnet/environments/hosted-grafana-cd/`

**Wave Deploy Testing**:

Bench runs during hosted Grafana deployments to validate releases:

**Test Suites** (from `rrc-bench-suites.libsonnet`):
1. **grafana-api-tests** - Core API smoke tests
2. **grafana-bench Playwright** - Browser tests
3. **logs-drilldown** - Logs feature tests
4. **slo-app** - SLO application tests

**Configuration**:
- Bench version: v1.0.3
- Prometheus metrics: Enabled
- Slack notifications: Per-team routing via CODEOWNERS
- Dashboard links: Argo Workflows dashboard

**Integration Point** (`deploy.libsonnet`):
```jsonnet
runBenchTest(step_name, instanceURL, suite)::
  suiteConfig.benchStep(step_name)
    .withBenchTest(instanceURL, suite)
    .buildStep(useClusterWorkflowTemplate=false)
```

### 6. Other Deployment Uses

**Unified Storage CD**:
- Tests against 4 RRC versions (instant, fast, steady, slow)
- Suite: `grafana-core/playlists` from grafana-api-tests
- Validates unified storage deployment

**Federal GE Grafana**:
- Post-install validation
- Runs as Helm hook
- Validates deployment after upgrades

---

## Observability & Monitoring

### 1. Three-Layer Observability Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Logs** | Loki | Workflow execution logs, debug info |
| **Metrics** | Prometheus | Test counts, durations, trends |
| **Traces** | Tempo | Distributed trace analysis |

### 2. CI Log Flow (GitHub Actions → Loki)

**Component**: `cicd-o11y` (OpenTelemetry Collector)

**Flow**:
```
GitHub Actions Webhook
  ↓
OTel Collector (cicd-o11y)
  - Receiver: githubactions (webhook on :3333/github/webhook)
  - Processors: batch, resource/logs, filter
  ↓
Exporters (parallel):
  - Loki (ops, dev, singlewave)
  - Tempo (distributed traces)
  - Prometheus (spanmetrics)
```

**Location**: `deployment_tools/ksonnet/lib/cicd-o11y/`

**Configuration Files**:
- `config-base.libsonnet` - Core pipeline
- `config-ops.libsonnet` - Ops environment
- `cicd-o11y.libsonnet` - Kubernetes deployment

**Resource Labels**:
- `ci.github.workflow.run.path` - Workflow file path
- `ci.github.workflow.run.event` - Event type (push, PR)
- `service.namespace` - Set to "cicd-o11y"
- `scm.git.repo` - Repository name

**Loki Credentials**:
- Main: `LOKI_ENDPOINT`, `LOKI_USERNAME`, `LOKI_PASSWORD`
- Dev: `LOKI_DEV_ENDPOINT` (dev environments only)
- Singlewave: `LOKI_SINGLEWAVE_ENDPOINT`

### 3. Metrics Collection

**Bench → Prometheus**:
```
Bench CLI
  ↓
PrometheusReporter (pkg/reporter/prometheus_reporter.go)
  ↓
Protocol Buffers Remote Write
  ↓
Prometheus Push Endpoint (ops-03-ops-eu-south-0)
```

**Endpoint**: `https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push`

**Authentication**:
- User ID: 10428
- Password: Token-based via `PROMETHEUS_PASSWORD`

**Labels Applied**:
```
job="bench"
grafana_version="{version}"
suite_run="{run_name}"
status="{passed|failed}"
{custom attributes from --run-attribute}
```

### 4. Dashboard Architecture

**Template-Based System**:

Bench doesn't create dashboards itself. Instead:
- Teams create dashboards in ops Grafana
- Bench provides dashboard URL via `--run-dashboard` with template variables
- URLs included in Slack notifications

**Dashboard URL Template** (`pkg/dashboard/dashboard.go`):
```
Template: http://grafana.com/d/dashboard?run={{.Id}}
Rendered: http://grafana.com/d/dashboard?run=abc123
```

**Common Dashboard Patterns**:
- Filter by `suite_run` label
- Visualize test trends over time
- Alert on test failure rates
- Track test duration trends

### 5. Slack Notification Flow

**Architecture**:
```
Test Execution
  ↓
Result Parsing
  ↓
CODEOWNERS File (maps files → teams)
  ↓
CodeownersMapping (maps teams → Slack channels)
  ↓
SlackNotifier (pkg/notifier/slack_notifier.go)
  ↓
Slack API (formatted blocks with dashboard link)
```

**CodeownersMapping Format** (`codeowners-mapping.yaml`):
```yaml
codeowners:
  - github_team: "@grafana/backend-services"
    slack_channel_id: "C123456"
    slack_channel: "#backend-services"
```

**Notification Content**:
- Suite run ID and status
- Per-test results (file, duration, status)
- Dashboard link (if configured)
- Test failure details

---

## Key Integration Points

### 1. Version Management

**Bench Versions**:
- Production: `v1.0.3` (semantic versioning)
- Development: `experimental` / `dev-{commit}` (latest main)

**Image Registry**:
- `ghcr.io/grafana/`

**Image Variants**:
- `grafana-bench:{version}` - Standard (includes K6)
- `grafana-bench-playwright:{version}` - With Playwright + browsers

**Version Selection in Libsonnet**:
```jsonnet
local bench = import 'ksonnet/lib/bench/versions.libsonnet';
local benchV0_6_11 = bench.v0_6_11;  // Specific version
local benchLatest = bench.experimental;  // Latest dev
```

### 2. Secret Management

**Common Secrets Used**:
- `grafana-credentials` - Admin user/password
- `bench-credentials` - K6, GitHub, Prometheus, Slack tokens
- `grafana-credentials-unified-storage` - Unified storage specific

**Secret Injection**:
```jsonnet
// In Argo workflow
envFrom: [
  secretRef: { name: 'grafana-credentials' },
  secretRef: { name: 'bench-credentials' },
]
```

### 3. Test Suite Patterns

**Common Patterns**:

1. **Embedded Grafana Pattern** (grafana-bench-cd):
   - Sidecar Grafana container
   - Self-contained, no external deps
   - Health checks before test execution

2. **Instance URL Pattern** (hosted-grafana-cd):
   - Tests against deployed Grafana instance
   - Dynamic URL from deployment
   - Credentials from secrets

3. **RRC Validation Pattern** (unified-storage-cd):
   - Test multiple release channels
   - Validate rolling releases
   - Channel-specific configuration

4. **Post-Install Hook Pattern** (federal-ge-grafana):
   - Helm hook job
   - Validates deployment
   - Fails upgrade if tests fail

### 4. Error Handling Patterns

**noFail Option**:
```jsonnet
suite {
  noFail: true,  // Test failure doesn't fail workflow
}
```

**Use Cases**:
- Playwright tests (browser tests can be flaky)
- Non-critical test suites
- Gradual rollout of new tests

**Exit Code Handling**:
- Bench returns exit code 1 on test failure
- Argo workflow fails step if exit code != 0
- `noFail: true` ignores non-zero exit codes

---

## File Locations

### Bench Repository

**Main Codebase**: `/Users/jeff/projects/bench-jalevin-cleanup-docs/`

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
```

### deployment_tools Repository

**Location**: `/Users/jeff/projects/deployment_tools/`

```
ksonnet/lib/
  bench/
    versions.libsonnet            # Version registry
    template.libsonnet            # Pre-configured builder
    v1.0.3/main.libsonnet        # Production version
    experimental/main.libsonnet   # Development version

  argo-workflows-util/
    common-steps/
      bench-v2.libsonnet          # Modern Argo integration
      bench.libsonnet             # Legacy Argo integration
    utils/
      templates.libsonnet         # bench_test template

  cicd-o11y/
    config-base.libsonnet         # OTel collector config
    config-ops.libsonnet          # Ops environment
    cicd-o11y.libsonnet           # Kubernetes deployment

ksonnet/environments/
  grafana-bench-cd/
    main.jsonnet                  # Self-testing orchestrator
    bench-test-suites.libsonnet   # Test suite definitions

  hosted-grafana-cd/
    rrc-bench-suites.libsonnet    # RRC test suites
    deploy.libsonnet              # Deployment integration

  unified-storage-cd/
    deploy.libsonnet              # Unified storage testing

  federal-ge-grafana/
    deployment-validation.libsonnet # Post-install validation
```

---

## Summary

Grafana Bench provides a unified testing platform with three key integration layers:

1. **CLI Layer**: Flexible command-line tool with multi-framework support
2. **Deployment Layer**: Libsonnet integration for Argo Workflows and Kubernetes
3. **Observability Layer**: Logs (Loki), Metrics (Prometheus), Traces (Tempo)

**Key Strengths**:
- Portable tests across environments
- Standardized reporting and observability
- Team-based notification routing
- Extensible architecture
- Production-proven in Grafana deployment pipelines

**Primary Use Cases**:
- CI/CD validation (GitHub Actions)
- Deployment validation (Argo Workflows)
- Release channel testing (Wave deploys)
- Continuous monitoring (Post-deploy smoke tests)

This architecture enables Grafana teams to write tests once and run them everywhere with consistent observability and reporting.
