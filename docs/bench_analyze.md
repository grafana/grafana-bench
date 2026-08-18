## bench analyze

analyze testRun events in Loki and emit defectConfirmed events

### Synopsis


analyze subcommand queries Loki for msg=testRun events and emits msg=defectConfirmed
events for (service, runStage, testFile, version) tuples that breach the
defect-confirmation rule.

The version axis is selected by --analyze-version-axis: 'grafanaVersion'
(default) bisects on the Grafana build under test (the RRC/core path), while
'serviceVersion' bisects on the data source plugin version carried from
--service-version, which moves independently of the Grafana version.

Rule (v2):
  1. Persistence:       ≥ --analyze-min-failures consecutive failing testRun
                        events on the target version.
  2. Regression boundary: the most recent prior distinct version in the same
                        runStage has ≥ --analyze-min-prior-passing passing
                        runs and 0 failures.
  3. Deterministic signature: the canonicalised exitMessage is stable across
                        the failing tail. Mismatches emit confidence=suspected.
  4. Retry evidence:    when retries are enabled (maxAttempts > 1), each failure
                        in the tail must have burned its full retry budget
                        (attempts == maxAttempts) to stay confidence=confirmed;
                        otherwise confidence=suspected. Runs without retry data
                        are unaffected. retryExhausted is emitted either way.

Events are emitted to stdout in the same logfmt/JSON shape as other bench
events (tool=bench, service=<svc>, msg=defectConfirmed, …) so they ship into
Loki identically and can be queried alongside msg=testRun / msg=suiteRun.

Credentials can be passed via flags or the BENCH_LOKI_URL / BENCH_LOKI_USERNAME /
BENCH_LOKI_PASSWORD / BENCH_LOKI_BEARER_TOKEN environment variables.


```
bench analyze [flags]
```

### Examples

```

# dry-run over the last 72h for a plugin:
grafana-bench analyze \
  --analyze-loki-url https://logs-prod-xxx.grafana.net \
  --analyze-service grafana-athena-datasource \
  --analyze-window 72h \
  --analyze-emit=false \
  --log-level DEBUG

# real run, scoped to the CI stream:
grafana-bench analyze \
  --analyze-loki-url https://logs-prod-xxx.grafana.net \
  --analyze-loki-bearer-token "$BENCH_LOKI_BEARER_TOKEN" \
  --analyze-service grafana-athena-datasource \
  --analyze-run-stage ci

# bisect on the plugin version instead of the Grafana version:
grafana-bench analyze \
  --analyze-loki-url https://logs-prod-xxx.grafana.net \
  --analyze-service grafana-clickhouse-datasource \
  --analyze-version-axis serviceVersion \
  --analyze-run-stage ci

```

### Options

```
      --analyze-emit                       Emit msg=defectConfirmed events on stdout. Set false for dry-run. (default true)
      --analyze-loki-bearer-token string   Bearer token for Loki. Overridden by BENCH_LOKI_BEARER_TOKEN.
                                           If set, takes precedence over basic auth.
      --analyze-loki-password string       Basic-auth password for Loki. Overridden by BENCH_LOKI_PASSWORD.
      --analyze-loki-selector string       LogQL stream selector for testRun events. Override for RRC/cluster context. (default "{service_name=~\".+\"} |= \"tool=bench\" |= \"msg=testRun\"")
      --analyze-loki-url string            REQUIRED. Base URL for the Loki API (e.g. https://logs-prod-xxx.grafana.net).
                                           Overridden by the BENCH_LOKI_URL environment variable.
      --analyze-loki-username string       Basic-auth username for Loki. Overridden by BENCH_LOKI_USERNAME.
      --analyze-min-failures int           Persistence threshold: consecutive failing testRun events required to confirm a defect. (default 3)
      --analyze-min-prior-passing int      Regression-boundary threshold: passing runs required on the most recent prior distinct version on the analysis axis. (default 2)
      --analyze-output string              Output format for emitted defectConfirmed events ('json' or 'text'). (default "json")
      --analyze-run-stage string           Restrict analysis to a single runStage. Empty evaluates each observed stage independently.
      --analyze-service string             REQUIRED. Restrict analysis to testRun records with this service value.
      --analyze-version-axis string        Version axis the rule groups and bisects on: 'grafanaVersion' (RRC/core Grafana builds)
                                           or 'serviceVersion' (the data source plugin version carried from --service-version). (default "grafanaVersion")
      --analyze-window duration            Lookback window for the Loki query_range request. (default 24h0m0s)
  -h, --help                               help for analyze
```

### Options inherited from parent commands

```
      --config string      path to config file (default "bench.yaml")
      --env string         path to a file with the environment variables.
                           If none is specified and a .env files exists in the work directory, it will be used
      --log-level string   set the log level ('ERROR', 'WARN', 'INFO', 'DEBUG').
                            overridden by the BENCH_LOG_LEVEL environment variable (default "ERROR")
```

### SEE ALSO

* [bench](bench.md)	 - grafana bench

###### Auto generated by spf13/cobra on 5-Aug-2026
