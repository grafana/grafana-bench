# Reporting metrics with Bench

Bench supports metrics for your test suites as well as custom metrics for
anything you may want to measure about Grafana.

## Default Bench metrics and labels

To enable metrics from bench, add the following flags:

* `--prometheus-metrics` - enables metrics
* `--prometheus-url` - the endpoint for your metrics
* `--prometheus-user`

and set `PROMETHEUS_PASSWORD` as an environment variable.

For example:

This example uses the test runner with default metrics.

```sh
PROMETHEUS_PASSWORD=MYSUPERSECRETPROMTOKEN grafana-bench test 
--test-runner k6 \
        --suite-path myMetricsTest.ts \
        --prometheus-metrics \
        --prometheus-strict-lint \
        --prometheus-url "https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push" \
        --prometheus-user 10428 
```

### Default Metrics

Built-in metrics are prefixed with `bench_` and describe your test suite.

bench_tests_error
bench_tests_executed
bench_tests_failed
bench_tests_flakey
bench_tests_passed
bench_total_duration_seconds

### Default Labels

Built-in labels describe the known details about Grafana

grafana_version
suite_run
status (failed, passed)

## Custom Metrics

Custom metrics are good for recording how a measurement of your service or domain
changes over time.

A couple of examples:

* frontend asset bundle size
* frontend render time
* request latency at 100vu load

## Metric Names and Labels

It is recommended to read the prometheus conventions for [metric and label naming](https://prometheus.io/docs/practices/naming/).
This helps us keep metrics useful and discoverable.

### General rules

Metrics should look like `{AREA}_{TYPE}_{SUBJECT}_{INCREMENT}`. For instance:

`fe_perf_used_js_heap_size_bytes`

`be_perf_latency_seconds`

Do not include the name of the service as that will be applied as a label, allowing
for querying metrics by service.

* Metric names and labels should be snake_case
* Bench generates its own metrics for test suites in the form `bench_tests_executed`.
* Custom metrics do not have the `bench_` prefix.
* Use base units - bytes instead of megabytes, seconds instead of milliseconds, etc
* Suffix describes the unit - http_request_duration_**seconds**

The `--prometheus-strict-lint` lints metric and label names for to ensure they
 adhere to proper conventions. If your command fails linting, we will not shop
 the metrics.

### Labels on custom metrics

Bench handles injecting labels for grafana url, grafana version, and bench. You can add your
own labels in the text exposition format, however, you should not override them yourself
and should instead pass those to the bench CLI.

## Passing cusstom metrics to bench

To report custom metrics, first write them to a file in the text exposition format
then pass them to the bench reporter.

You can create custom metrics with cli flags or write them to a file in
the prometheus text exposition format and passing that into the bench reporter.
We recommend the latter.

### Text exposition format example

```text
http_requests_total{method="GET",path="/"} 100
http_requests_total{method="POST",path="/api"} 50 1678886400000
cpu_usage_seconds_total 123.45
```

## Generating an ops metrics key

### For GitHub Actions (Recommended)

If you're running bench in a GitHub Actions workflow, use the `setup-grafana-bench` action which automatically fetches shared Prometheus secrets from Vault. See the [GitHub Actions documentation](github_actions.md#authentication-and-ci-tokens) for details.

The action sets up these environment variables for you:
- `PROMETHEUS_URL`: <https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push>
- `PROMETHEUS_USER`: 10428
- `PROMETHEUS_PASSWORD`: Fetched from shared Vault secret

Simply add the `--prometheus-metrics` flag to your bench commands after the setup step if you're calling bench directly. If you are running the bench container, pass those in as environment variables to the docker command.

### For Other Use Cases

If you need to create your own token for local development or non-GitHub Action environments:

1. Go to the Cloud Access Policy app: <https://ops.grafana-ops.net/a/grafana-auth-app>
2. Create a new access policy with metrics:write permissions
3. Name it after your app/domain/namespace
4. Click create
5. Create a token and copy it
6. Add the token to vault (requires access to the deployment_tools repository):
   ```bash
   # From the root of the deployment_tools repository
   # First, request timed access to vault
   make timed-access-cli request-access

   # Then add the token to vault
   VAULT_INSTANCE=prod ./scripts/vault/vault-put secret/{namespace}/prometheus_token prometheus_token={token}
   ```
7. Use the token as the PROMETHEUS_PASSWORD environment variable with these additional settings:
   - PROMETHEUS_URL: <https://prometheus-ops-03-ops-eu-south-0.grafana-ops.net/api/prom/push>
   - PROMETHEUS_USER: 10428

## Complete example

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
