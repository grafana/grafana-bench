# TruffleHog JSON Parser for Grafana Bench

This parser allows you to report TruffleHog secret-scanning results (native JSON format) through Grafana Bench's `report` command.

## Overview

The integration model is the same as [zizmor](https://github.com/grafana/grafana-bench/tree/main/pkg/executor/zizmor): each repo has TruffleHog integrated (e.g. in CI). The scan produces a JSON output file (e.g. `result.json` or `results.json`); Bench parses that file via `report --report-input trufflehog` and sends metrics/logs for dashboards and alerting.

TruffleHog finds secrets in code and history and outputs a JSON array of findings. This parser reads that JSON to generate metrics and structured logs.

**Security:** The parser never sends credentials or secret material. It only uses detector name, generic detector description, file path, line number, and verified/unverified flag. The `Raw`, `RawV2`, `Redacted`, and `ExtraData` fields from the JSON are never included in report output, metrics, or logs.

## Usage

### Basic Command

```bash
# Run TruffleHog and save JSON (per-repo CI typically produces result.json or results.json)
trufflehog filesystem . --json > result.json

# Report results with grafana-bench (parses the file; no credentials are sent)
grafana-bench report \
  --report-input trufflehog \
  --report-output text \
  --service trufflehog \
  --service-version 3.0.0 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  result.json
```

### With Structured Logs

```bash
grafana-bench report \
  --report-input trufflehog \
  --report-output log \
  --service trufflehog \
  --service-version 3.0.0 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  results.json
```

### With Prometheus Metrics

```bash
grafana-bench report \
  --report-input trufflehog \
  --report-output log \
  --service trufflehog \
  --service-version 3.0.0 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  --prometheus-metrics \
  --prometheus-url https://prometheus.example.com/api/v1/write \
  results.json
```

## Metrics Tracked

The TruffleHog parser tracks the following as suite-level attributes and Prometheus metrics:

- **total_findings**: Total number of secrets/findings reported
- **verified**: Count of verified findings
- **unverified**: Count of unverified findings
- **unique_detectors**: Number of distinct detector types (e.g. Github, AWS, Private Key)

Each finding is reported as a test run with:
- **detector**: Detector name (e.g. "Github", "AWS")
- **verified**: Whether the finding was verified
- **testFile**: Source file (from Filesystem, Git, or GitHub metadata when present)
- **exitMessage**: Detector description and line number when available

## Query Examples

With structured logs and Prometheus metrics:

```promql
# Total findings per repo
trufflehog_total_findings{repo="my-org/my-repo"}

# Verified vs unverified over time
trufflehog_verified{repo="my-org/my-repo"}
trufflehog_unverified{repo="my-org/my-repo"}

# Findings by detector type
trufflehog_detector_findings{detector="Github", repo="my-org/my-repo"}

# Repos that have run a scan (including zero findings)
trufflehog_scan_completed
```

## Grafana dashboards: metrics you can build

Bench injects a `repo` label on every metric (from `--suite-name`). Use these PromQL ideas for the panels you want.

| Goal | Panel type | PromQL (or approach) |
|------|------------|----------------------|
| **Secrets found per week / month** | Time series or stat | `sum(trufflehog_total_findings)` over time. For “total in last 30 days”, set dashboard time range to last 30d and use `sum(trufflehog_total_findings)` as a stat. For weekly/monthly buckets, use Grafana’s “Group by” (e.g. by week) on a time series of `sum(trufflehog_total_findings)`. |
| **Which repos leak the most secrets** | Table or bar chart | `topk(20, sum by (repo) (trufflehog_total_findings))` — repos with highest total findings. Sort descending. |
| **How many verified real secrets** | Stat or time series | `sum(trufflehog_verified)` — total verified findings across all repos. Per repo: `sum by (repo) (trufflehog_verified)`. |
| **Secret types breakdown** | Pie chart or bar chart | `sum by (detector) (trufflehog_detector_findings)` — count by secret type (e.g. Github, AWS, Private Key). |
| **TruffleHog repo coverage** | Stat | `count(count by (repo) (trufflehog_scan_completed))` — number of distinct repos that have run TruffleHog and reported at least once (including zero findings). |

Metrics are gauges: each report run sends a snapshot (e.g. “this repo had 5 findings at scan time”). For “secrets per week/month”, use the dashboard time range and the queries above; for trend over time, plot `sum(trufflehog_total_findings)` or `sum(trufflehog_verified)` in a time series.

## JSON Format

The parser expects TruffleHog's native JSON output: a top-level array of finding objects. Example (minimal):

```json
[
  {
    "SourceMetadata": {
      "Data": {
        "Filesystem": {
          "file": "path/to/file",
          "line": 10
        }
      }
    },
    "DetectorName": "Github",
    "DetectorDescription": "GitHub personal access tokens...",
    "Verified": true
  }
]
```

Supported source locations: `SourceMetadata.Data.Filesystem`, `SourceMetadata.Data.Git`, and `SourceMetadata.Data.GitHub` (each with optional `file` and `line`). An empty array `[]` is valid and indicates no findings.

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Secret Scan
on: [push, pull_request]

jobs:
  trufflehog-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run TruffleHog
        uses: trufflesecurity/trufflehog-actions-scan@v1
        with:
          path: .
          output: results.json
        continue-on-error: true

      - name: Report with Grafana Bench
        env:
          PROMETHEUS_URL: ${{ secrets.PROMETHEUS_URL }}
        run: |
          grafana-bench report \
            --report-input trufflehog \
            --report-output log \
            --service trufflehog \
            --suite-name ${{ github.repository }} \
            --run-stage ci \
            --prometheus-metrics \
            results.json
```

Or with the TruffleHog CLI directly:

```yaml
      - name: Install TruffleHog
        run: |
          go install github.com/trufflesecurity/trufflehog/v3@latest
      - name: Run TruffleHog
        run: trufflehog filesystem . --json > results.json || true
      - name: Report with Grafana Bench
        run: |
          grafana-bench report \
            --report-input trufflehog \
            --report-output log \
            --service trufflehog \
            --suite-name ${{ github.repository }} \
            --run-stage ci \
            --prometheus-metrics \
            results.json
```

## Configuration File

You can use a bench.yaml configuration file:

```yaml
service:
  name: "trufflehog"
  version: "3.0.0"

suite:
  name: "my-org/my-repo"

report:
  input: "trufflehog"
  output: "log"

prometheus:
  metrics: true
  url: "https://prometheus.example.com/api/v1/write"

run:
  stage: "ci"
```

Then run:

```bash
trufflehog filesystem . --json > results.json
grafana-bench report --config bench.yaml results.json
```

## Exit Codes

- **0**: Success - results reported successfully
- **1**: Error - failed to parse or report results

The report command's exit code reflects reporting success, not whether findings were present. You can always report results even when the scan finds secrets.
