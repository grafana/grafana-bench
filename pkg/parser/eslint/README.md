# pkg/parser/eslint

Parser for aggregated ESLint suppression counts, grouped by codeowner.

## Input format

A JSON file produced by `npm run lint:suppressions:by-codeowner`
(see [grafana/grafana#119446](https://github.com/grafana/grafana/pull/119446)):

```json
{
  "summary": [
    { "codeowner": "@grafana/dashboards-squad", "suppressions": 494 },
    { "codeowner": "@grafana/grafana-frontend-platform", "suppressions": 465 },
    { "codeowner": "@grafana/grafana-fdataviz", "suppressions": 345 }
  ]
}
```

### Field rules

| Field | Required | Behaviour when absent |
|-------|----------|-----------------------|
| `summary` | yes | parse error |
| `codeowner` | yes | parse error |
| `suppressions` | no | entry skipped silently, no metric emitted |

## Metrics emitted

This parser emits no Prometheus metrics. It validates the report structure and
populates the `SuiteRunSummary` attributes (e.g. `package`) for use by
reporters and loggers.

## CLI usage

```bash
# Generate the suppressions report in your CI/CD workflow
npm run lint:suppressions:by-codeowner

# Report results with grafana-bench
grafana-bench report \
  --report-input eslint \
  --report-output log \
  --service eslint \
  --service-version 11.4.0 \
  --suite-name grafana/grafana \
  --js-coverage-package "@grafana/grafana" \
  --run-stage ci \
  --prometheus-metrics \
  eslint-report.json
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--js-coverage-package` | yes | Value for the `package` label on all metrics |
| `--suite-name` | yes | Injected as the `repo` label by `cmd/report` |
