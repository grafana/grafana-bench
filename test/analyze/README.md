# `bench analyze` smoke environment

A disposable, real-Loki playground for validating `grafana-bench analyze` end-to-end. Use when you need confidence that the analyzer round-trips through a genuine Loki `query_range` API — the in-process tests in [cmd/analyze/command_test.go](../../cmd/analyze/command_test.go) cover the rule engine, but they talk to an `httptest.Server` fake, not a real ingester.

## What it does

1. Brings up a single-node Loki (filesystem storage, no auth) on `localhost:3100`.
2. Pushes a fixture that replays the 2026-03-26 `permissions.ts` regression: 3 green runs on the last-known-good build, 4 failing runs on the first-bad build, with varying `pid` in the exit message.
3. Runs `grafana-bench analyze` against that Loki.
4. Expected output: **exactly one** `msg=defectConfirmed` line on stdout, with `confidence=confirmed`, `grafanaVersion=13.0.0-23542128402`, `priorPassingVersion=13.0.0-23563050832`.

## Requirements

- Docker (or compatible, e.g. OrbStack)
- Go 1.22+
- `grafana-bench` on `PATH`, or running via `go run .`

## One-shot run

From the repo root:

```sh
make analyze-smoke
```

That target:
- starts the Loki container (waits for `/ready`)
- runs the seeder
- invokes `go run . analyze` against the seeded data
- tears the container down

## Manual workflow

If you want to poke at the stream yourself (e.g. with `logcli` or Grafana Explore):

```sh
# 1. Start Loki
docker compose -f test/analyze/docker-compose.yml up -d

# 2. Seed fixtures
go run ./test/analyze/seed -loki http://localhost:3100

# 3. Run the analyzer
go run . analyze \
  --analyze-loki-url http://localhost:3100 \
  --analyze-service grafana-pro \
  --analyze-run-stage ci \
  --analyze-window 24h

# 4. Clean up when done
docker compose -f test/analyze/docker-compose.yml down -v
```

## Seeder flags

`go run ./test/analyze/seed -h`:

| Flag | Default |
|---|---|
| `-loki` | `http://localhost:3100` |
| `-service` | `grafana-pro` |
| `-run-stage` | `ci` |
| `-test-file` | `permissions.ts` |
| `-good-version` | `13.0.0-23563050832` |
| `-bad-version` | `13.0.0-23542128402` |

All timestamps are anchored to `time.Now()` so the analyzer's default 24h window always covers them.

## Troubleshooting

**`loki at http://localhost:3100 not ready after 60s`** — Docker Desktop or OrbStack isn't running, or port 3100 is already in use. `lsof -i :3100` to check.

**Analyzer emits zero events** — confirm the push actually landed:
```sh
curl -sG 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={service_name="grafana-pro"} |= "msg=testRun"' \
  --data-urlencode "start=$(date -v-2H +%s)000000000" \
  --data-urlencode "end=$(date +%s)000000000" | jq '.data.result[0].values | length'
```
If that returns `0`, the seed step failed or Loki hasn't flushed yet (unusual with filesystem storage).
