# Bench metrics

Bench supports metrics for anything you may want to measure about Grafana.

## Default Bench metrics and labels

When you enable metrics, bench emits metrics about the test suite. They are prefixed with `bench_`

### Default Metrics

bench_tests_error
bench_tests_executed
bench_tests_failed
bench_tests_flakey
bench_tests_passed
bench_total_duration_seconds

### Default Labels

grafana_version
suite_run
status (failed, passed)

## Emitting metrics

You can emit metrics when using the bench reporter or using the bench test runner.
The following is an example using the bench test runner.

```sh
PROMETHEUS_PASSWORD=MYSUPERSECRETPROMTOKEN grafana-bench test 
    --test-runner k6 \
    --suite-path myMetricsTest.ts \
    --prometheus-metrics \
    --prometheus-strict-lint \
    --prometheus-url "https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push" \
    --prometheus-user 10428 
```

## Metric Names and Labels

It is recommended to read the prometheus conventions for [metric and label naming](https://prometheus.io/docs/practices/naming/).
This helps us keep metrics useful and discoverable.

The `--prometheus-strict-lint` lints metric and label names for to ensure they
 adhere to proper conventions. If your command fails linting, we will not shop
 the metrics.

## Custom metrics

You can create custom metrics with cli flags or writing them to a file in
the prometheus text exposition format and passing that into the bench reporter.
We recommend the latter.

Text exposition format exmaple:

```text
http_requests_total{method="GET",path="/"} 100
http_requests_total{method="POST",path="/api"} 50 1678886400000
cpu_usage_seconds_total 123.45
```

The following is a complete example using a github action to run a playwright test
and then pass the playwright output json and metrics file to the bench reporter.

```yaml
name: Measure frontend asset size

on:
  schedule:
    - cron: '0 * * * *'
  # Optional: Also allow manual triggering
  workflow_dispatch:

jobs:
  measure-frontend-assets:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write

    steps:
      - name: checkout code
        uses: actions/checkout@v4
        with:
          persist-credentials: false
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
          cache: false

      - name: install bench
        run: go install

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'yarn'
          cache-dependency-path: './CI/plugin-e2e/yarn.lock'

      - name: Cache Playwright binaries
        uses: actions/cache@v4
        id: playwright-cache
        with:
          path: '~/.cache/ms-playwright'
          key: '${{ runner.os }}-playwright'
          restore-keys: |
            ${{ runner.os }}-playwright-

      - name: setup playwright
        #if: steps.playwright-cache.outputs.cache-hit != 'true'
        working-directory: ./CI/plugin-e2e
        run: |
          yarn install && yarn playwright install

      - name: run playwright tests
        working-directory: ./CI/plugin-e2e
        env:
          PLAYWRIGHT_JSON_OUTPUT_NAME: /tmp/results.json
        run: yarn run test --grep @performance --reporter json

      #- name: cat results
      #  run: |
      #    cat /tmp/results.json
      #    cat /tmp/asset-metrics.txt

      - id: get-secrets
        uses: grafana/shared-workflows/actions/get-vault-secrets@get-vault-secrets-v1.2.0
        with:
          # Secrets placed in ci/repo/grafana/grafana-bench vault
          repo_secrets: |
            PROMETHEUS_PASSWORD=prometheus_token:prometheus_token

      - name: report result
        env:
          PROMETHEUS_URL: "https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push"
          PROMETHEUS_USER: 10428
        run: |
          grafana-bench report \
            --grafana-url "https://leeoniya.grafana.net" \
            --grafana-version "rrc-instant" \
            --test-suite-name "FrontendAssetSize" \
            --report-input playwright \
            --report-output log \
            --prometheus-metrics \
            --run-metrics-file "/tmp/asset-metrics.txt" \
            --log-level DEBUG \
            /tmp/results.json
```
