# Zizmor SARIF Parser for Grafana Bench

This parser allows you to report zizmor security scan results in SARIF format through Grafana Bench's `report` command.

## Overview

Zizmor is a security tool that analyzes GitHub Actions workflows for security vulnerabilities. It outputs results in SARIF (Static Analysis Results Interchange Format), which this parser can read to generate metrics and logs.

## Usage

### Basic Command

```bash
# Run zizmor and save results
zizmor --format sarif . > results.sarif

# Report results with grafana-bench
grafana-bench report \
  --report-input zizmor \
  --report-output text \
  --service zizmor \
  --service-version 1.14.2 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  results.sarif
```

### With Structured Logs

```bash
grafana-bench report \
  --report-input zizmor \
  --report-output log \
  --service zizmor \
  --service-version 1.14.2 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  results.sarif
```

### With Prometheus Metrics

```bash
grafana-bench report \
  --report-input zizmor \
  --report-output log \
  --service zizmor \
  --service-version 1.14.2 \
  --suite-name my-org/my-repo \
  --run-stage ci \
  --prometheus-metrics \
  --prometheus-url https://prometheus.example.com/api/v1/write \
  results.sarif
```

## Metrics Tracked

The zizmor parser tracks the following metrics as suite-level attributes:

- **total_vulnerabilities**: Total number of security issues found
- **high_severity**: Number of high-severity (error) findings
- **medium_severity**: Number of medium-severity (warning) findings
- **low_severity**: Number of low-severity (note) findings
- **unique_rules_violated**: Number of unique security rules violated
- **tool_version**: Version of zizmor used

Each finding is tracked as a test run with:
- **ruleId**: The security rule that was violated (e.g., "artipacked", "template-injection")
- **severity**: The severity level (error, warning, note)
- **testFile**: The workflow file where the issue was found
- **exitMessage**: Description of the issue with line number

## Query Examples

With structured logs in Prometheus/Loki, you can query:

```promql
# Count vulnerabilities per repo
count by (suiteName) (testRun{service="zizmor", status="failed"})

# Track high-severity issues over time
count by (suiteName) (testRun{service="zizmor", severity="error"})

# Most common rule violations
count by (ruleId) (testRun{service="zizmor"})

# Repos with dangerous-triggers violations
testRun{service="zizmor", ruleId="dangerous-triggers"}
```

## SARIF Format

The parser expects SARIF 2.1.0 format as output by zizmor. Example:

```json
{
  "$schema": "https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/schemas/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "zizmor",
          "version": "1.14.2"
        }
      },
      "results": [
        {
          "ruleId": "artipacked",
          "level": "error",
          "message": {
            "text": "Workflow uses artifacts without integrity validation"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": ".github/workflows/ci.yml"
                },
                "region": {
                  "startLine": 42
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Security Scan
on: [push, pull_request]

jobs:
  zizmor-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install zizmor
        run: |
          curl -sSL https://github.com/zizmorcore/zizmor/releases/download/v1.14.2/zizmor-linux-x86_64 -o /usr/local/bin/zizmor
          chmod +x /usr/local/bin/zizmor

      - name: Run zizmor scan
        run: |
          zizmor --format sarif . > results.sarif || true

      - name: Report with Grafana Bench
        env:
          PROMETHEUS_URL: ${{ secrets.PROMETHEUS_URL }}
        run: |
          grafana-bench report \
            --report-input zizmor \
            --report-output log \
            --service zizmor \
            --service-version 1.14.2 \
            --suite-name ${{ github.repository }} \
            --run-stage ci \
            --prometheus-metrics \
            results.sarif
```

### GitLab CI Example

```yaml
zizmor-scan:
  stage: security
  image: ubuntu:latest
  script:
    # Install zizmor
    - curl -sSL https://github.com/zizmorcore/zizmor/releases/download/v1.14.2/zizmor-linux-x86_64 -o /usr/local/bin/zizmor
    - chmod +x /usr/local/bin/zizmor

    # Run scan
    - zizmor --format sarif . > results.sarif || true

    # Report with bench
    - |
      grafana-bench report \
        --report-input zizmor \
        --report-output log \
        --service zizmor \
        --service-version 1.14.2 \
        --suite-name $CI_PROJECT_PATH \
        --run-stage ci \
        --prometheus-metrics \
        --prometheus-url $PROMETHEUS_URL \
        results.sarif
  artifacts:
    paths:
      - results.sarif
    when: always
```

## Configuration File

You can also use a bench.yaml configuration file:

```yaml
service:
  name: "zizmor"
  version: "1.14.2"

suite:
  name: "my-org/my-repo"

report:
  input: "zizmor"
  output: "log"

prometheus:
  metrics: true
  url: "https://prometheus.example.com/api/v1/write"

run:
  stage: "ci"
```

Then run:

```bash
zizmor --format sarif . > results.sarif
grafana-bench report --config bench.yaml results.sarif
```

## Exit Codes

- **0**: Success - results reported successfully
- **1**: Error - failed to parse or report results

Note: The report command returns the exit code based on reporting success, not on security findings. This allows you to always report results even when vulnerabilities are found.
