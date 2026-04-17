# Detecting defects across runs with `bench analyze`

`bench analyze` turns the logs that bench already emits into a signal about whether a failing test has caught a real defect. It queries Loki for `msg=testRun` events over a time window, applies a three-part rule to each `(service, runStage, testFile, grafanaVersion)` tuple, and emits a new `msg=defectConfirmed` event on stdout for every tuple that breaches the rule.

This is the automated, pre-deploy counterpart to the `invest:tests` label on `grafana/support-escalations`: `invest:tests` says "a human decided this should have been caught by tests," and `msg=defectConfirmed` says "bench flagged this as a regression before a customer escalated it." Put together, they give Defect Escape Rate (DER) a precision side — *of the defects that escaped to customers, how many did bench catch first?*

---

## When to use it

Use `bench analyze` when you want to:

- **Answer "did bench see this coming?"** after a customer escalation — run the analyzer over the escalation window and check whether a `msg=defectConfirmed` for the same `grafanaVersion` already exists in Loki.
- **Feed DER dashboards** with an automated bench-emitted signal alongside the manual `invest:tests` tag.
- **Replace ad-hoc bisection runs** against `grafana-dev.net` slugs. The analyzer already has every `(version, outcome, exitMessage)` triple it needs in Loki; there is no reason to re-run suites to produce the timeline by hand.

Don't use it for:

- Inline suite-pass/fail decisions — that's what `bench test`'s exit code is for. The analyzer runs over historical data and deliberately waits for Loki ingestion.
- Flake triage at the test-file level — flaky tests fail the persistence rule by design and will not be confirmed.

---

## How the rule works

A `(service, runStage, testFile, grafanaVersion)` tuple is a **confirmed defect** only when all three conditions hold.

### 1. Persistence

The tail of the tuple's records contains at least `--analyze-min-failures` consecutive events with `status` in `{failed, error}`. Default: **3**.

The tail requirement — not "any 3 failures in the window" — prevents a historical cluster of failures from registering if the test has since come green.

### 2. Regression boundary

Walking back through prior `grafanaVersion`s **in the same `runStage`**, the most recent distinct version must have at least `--analyze-min-prior-passing` passing runs and **zero** failures. Default: **2**. If the prior version itself had any failures, the tuple doesn't clear the boundary and the defect is not confirmed.

Scoping by `runStage` is load-bearing: it prevents a clean CI stream from being contaminated by a persistently-failing `nightly` or `rrc` stream running on the same build.

### 3. Deterministic signature

For each failing run in the tail, the analyzer builds a signature:

```
sha256(testFile + "|" + canonicalize(exitMessage))[:12]
```

`canonicalize` strips variable fragments that would otherwise break signature stability — ISO-8601 timestamps, UUIDs, process IDs, URL query strings, absolute paths, long hex blobs, and decimal numbers ≥3 digits are all rewritten to placeholder tokens before hashing.

- All tail signatures identical → `confidence=confirmed`
- Signatures drift across the tail → `confidence=suspected`

`suspected` events are useful for triage (the test is persistently failing and there *is* a clean prior version, but the failure mode is shifting) but should not be counted as confirmed regressions in DER math without manual review.

---

## Emitted event schema

Every confirmed or suspected defect produces one `msg=defectConfirmed` line. It inherits the bench-standard `tool=bench` + `service=<svc>` fields and adds:

| Field | Description |
|---|---|
| `runStage` | CI/nightly/rrc/local — the scoping axis from rule (2) |
| `testFile` | Unit of regression |
| `testFolder` | Folder the test lives in |
| `grafanaVersion` | The version the rule fired on (the bad version) |
| `priorPassingVersion` | The most recent distinct prior grafanaVersion with ≥ `min_prior_passing` green runs (the last-known-good) |
| `grafanaSlug` | Cluster identity for RRC dashboard joins |
| `grafanaUrl` | Cluster URL |
| `signatureHash` | 12-hex signature used to dedupe across runs |
| `confidence` | `confirmed` or `suspected` |
| `confidenceRuns` | Number of consecutive failing runs observed |
| `priorPassingRuns` | Number of passing runs on the prior version |
| `exitMessageCanonical` | Canonicalised exit message (≤500 chars) |
| `exitMessageSample` | Raw exit message from one failure (≤200 chars) |
| `firstFailureTime` | RFC3339 timestamp of the first failure in the tail |
| `lastFailureTime` | RFC3339 timestamp of the last failure in the tail |
| `sourceRunIds` | Comma-separated `runId`s of the failures — traceable back to raw `testRun` events |
| `analyzeWindowSeconds` | Window the analyzer ran over |
| `analyzedAt` | When the analysis was performed |
| `ruleVersion` | `v1` — emitted so future rule changes can be retroactively segmented |

---

## Quickstart

Set your Loki credentials as environment variables (the analyzer's flags fall back to them):

```sh
export BENCH_LOKI_URL="https://logs-prod-xxx.grafana.net"
export BENCH_LOKI_BEARER_TOKEN="glc_xxx"
# OR for basic auth:
# export BENCH_LOKI_USERNAME="123456"
# export BENCH_LOKI_PASSWORD="glc_xxx"
```

### Dry-run first

Always start with `--analyze-emit=false` and `--log-level DEBUG`. The analyzer logs the record counts it pulled and a per-defect summary on stderr, without emitting anything to stdout.

```sh
grafana-bench analyze \
  --analyze-service grafana-athena-datasource \
  --analyze-window 72h \
  --analyze-emit=false \
  --log-level DEBUG
```

Sample debug output:

```text
time=... level=DEBUG msg="querying loki for testRun events" selector="{service_name=~\".+\"} |= \"tool=bench\" |= \"msg=testRun\"" start=... end=...
time=... level=DEBUG msg="loaded testRun records" pulled=184 after_filter=41
time=... level=DEBUG msg="analysis complete" defects=1
time=... level=INFO  msg="dry-run defect" testFile=permissions.ts grafanaVersion=13.0.0-23542128402 priorPassingVersion=13.0.0-23563050832 confidence=confirmed signatureHash=a9c0f1b2d345 confidenceRuns=4
```

### Real emission

Drop `--analyze-emit=false` and let the analyzer write JSON events to stdout:

```sh
grafana-bench analyze \
  --analyze-service grafana-athena-datasource \
  --analyze-run-stage ci
```

Each emitted line is a self-contained JSON object in the `msg=defectConfirmed` schema above.

### Ship events back into Loki

Because events land on stdout in the same shape as every other bench event, the existing bench log-shipping path (promtail / alloy / whatever is already scraping your bench container) picks them up automatically. To verify in Grafana Explore:

```logql
{service_name="grafana-athena-datasource"}
  |= "tool=bench"
  |= "msg=defectConfirmed"
  | logfmt
```

---

## Tuning knobs

| Flag | Default | Use when |
|---|---|---|
| `--analyze-min-failures` | `3` | Raise (e.g. `5`) for noisy test suites — buys precision at the cost of catching slow-burn regressions later. Lower (e.g. `2`) only for low-frequency/nightly streams where waiting for 3 is >24h. |
| `--analyze-min-prior-passing` | `2` | Raise to demand more confidence that the prior version was really green. Lower for streams with sparse per-version coverage. |
| `--analyze-window` | `24h` | Match to your test cadence. For RRC-style streams running every 5 minutes, `24h` is plenty; for nightly-only streams, `7d` is more appropriate. |
| `--analyze-run-stage` | *(empty)* | Empty evaluates each observed stage independently — safe default. Pin to a single stage when debugging or when a stream is noisier than others. |
| `--analyze-loki-selector` | `{service_name=~".+"} \|= "tool=bench" \|= "msg=testRun"` | Override to scope to the RRC cluster stream: `{job=~"hosted-grafana-cd/grafana.*"} \|= "tool=bench" \|= "msg=testRun"`. |

---

## Running in CI

The analyzer is safe to run as a cron job or a scheduled GitHub Actions workflow. It's idempotent: the same window analysed twice emits the same signatures, so dashboards dedupe naturally via `count_over_time`.

```yaml
# .github/workflows/bench-analyze.yml
name: bench defect analyzer

on:
  schedule:
    - cron: '0 */2 * * *' # every 2 hours

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - name: install bench
        run: go install github.com/grafana/grafana-bench@latest
      - name: analyze
        env:
          BENCH_LOKI_URL: ${{ secrets.BENCH_LOKI_URL }}
          BENCH_LOKI_BEARER_TOKEN: ${{ secrets.BENCH_LOKI_BEARER_TOKEN }}
        run: |
          grafana-bench analyze \
            --analyze-service grafana-athena-datasource \
            --analyze-run-stage ci \
            --analyze-window 6h \
            --log-level INFO
```

Pick `--analyze-window` to be a small multiple of the cron interval — `6h` for a 2-hour cron gives the analyzer a comfortable overlap to absorb late-arriving Loki ingestion without missing anything.

---

## Wiring into DER dashboards

The [Datasource Escalations dashboard](bench_test.md) (`datasource-esc-quality-v1`) already consumes `msg=suiteRun` events from the same Loki stream and joins them against plugin deploy audit events to compute DER. `msg=defectConfirmed` is a new series on that same stream, so no new data sources are required — just new panels.

Suggested additions:

- **A stat** showing bench-confirmed defects per deploy window, directly analogous to the existing `invest:tests` count:
  ```logql
  sum by () (
    count_over_time(
      {service_name="${data_source:text}"}
        |= "tool=bench"
        |= "msg=defectConfirmed"
      [${window_days}d]
    )
  )
  ```

- **Annotations** on the DER timeseries, one per `grafanaVersion`, so reviewers can see which deploys bench flagged before they escalated.

- **A drill-down row** showing the `sourceRunIds`, `signatureHash`, and `exitMessageSample` for any defect in the current window — enabling one-click navigation to the raw `msg=testRun` lines.

---

## Troubleshooting

**"No defects emitted but I expected one."**
Start with `--analyze-emit=false --log-level DEBUG` and check:
1. `pulled=N` — is the selector finding records at all? If `pulled=0`, your selector is wrong or the window is wrong.
2. `after_filter=N` — is your `--analyze-service` value the canonical slug? Typos here silently filter everything out.
3. The failing tail must be the **last** `min_failures` records for that `(testFile, grafanaVersion)`. A single green run at the tail resets the persistence check.
4. The prior version needs **zero failures**, not just "enough passes." Check for flakes on the supposed last-known-good version — they disqualify it.

**"Confidence is `suspected` but I think it's the same defect."**
The canonicaliser is conservative — it won't normalise fragments it doesn't recognise. Inspect the raw `exitMessage` samples in the `msg=testRun` events for the failing runs and check what's varying. If it's a new category (e.g. a test-framework version number, a hostname), open an issue so the canonicalisation table can learn it.

**"Loki returns 429."**
The analyzer backs off exponentially with jitter and retries up to 3 times by default. Raise `--analyze-window` chunking indirectly by splitting into multiple invocations (e.g. one per service) if you're running against a heavily-loaded Loki.

**"I'm seeing duplicate defects in the dashboard."**
Dashboards should count distinct `signatureHash` values within the dedupe window, not rows. Multiple analyzer runs over overlapping windows will re-emit the same signature — that's intentional for freshness but needs `by (signatureHash)` on the panel query.

**"`pulled=0` in DEBUG but I can see the records in Grafana Explore."**
The default `--analyze-loki-selector` is `{service_name=~".+"} |= "tool=bench" |= "msg=testRun"` — logfmt-style substrings that match bench's default `--report-output=log` format. If your bench is shipping JSON (`--report-output=json`), those substrings won't appear in the raw line and the selector filters everything out. Either re-configure bench to emit logfmt, or override the selector with JSON-matching substrings, e.g.:

```sh
--analyze-loki-selector '{service_name=~".+"} |= "\"tool\":\"bench\"" |= "\"msg\":\"testRun\""'
```

The analyzer's record decoder auto-detects JSON vs logfmt per line, so the only thing that needs aligning is the stream-level LogQL filter.

---

## Related pages

- [bench analyze reference](bench_analyze.md) — full flag reference
- [bench test reference](bench_test.md) — where `msg=testRun` events come from
- [Architecture](architecture.md) — internal structure
- [Metrics](metrics.md) — DER is computed from Prometheus counters; defectConfirmed is the Loki-side signal
- [`test/analyze/` smoke harness](https://github.com/grafana/grafana-bench/tree/main/test/analyze) — `make analyze-smoke` runs the analyzer end-to-end against a local Loki
