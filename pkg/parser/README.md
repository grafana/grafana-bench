# pkg/parser

This package contains parsers for tool output formats. Each sub-package reads
the output of a specific tool and converts it into a `SuiteRunSummary` that
bench can report, log, or push as Prometheus metrics.

Parsers live here — not in `pkg/executor/` — because bench does not need to
*run* the tool to parse its output. Tools like zizmor are invoked separately
and their output is passed to `grafana-bench report`. Executors (in
`pkg/executor/`) that *do* run subprocesses import from this package for the
parsing step.

## Writing a new parser

The recommended workflow:

1. **Collect 2–3 real output files** from the tool — ideally one with
   findings/failures, one clean, and one edge case (empty output, parse error,
   etc.). Drop them in `pkg/parser/<toolname>/testdata/`.

2. **Decide which metrics matter.** Write them down in plain text before
   touching any code. For example:

   ```
   - total findings (gauge)
   - findings by severity: high / medium / low (gauge, label: severity)
   - findings by rule (gauge, label: rule_id)
   - scan_completed = 1 always (used to track coverage — if the metric is
     absent, the repo was never scanned)
   ```

   Think about the questions you want to answer in Grafana:
   - Which repos have the most issues?
   - Which rules fire most often?
   - Is coverage increasing or decreasing over time?

3. **Point an LLM at the tool output + your metrics list + this codebase.**
   A prompt like the following works well:

   ```
   Here are 2 sample outputs from <tool> (see testdata/).
   Here are the metrics I want to track: <your list from step 2>.
   Here is an existing parser for reference: pkg/parser/zizmor/parser.go
   Here is the SuiteRunSummary type: pkg/executor/executor.go
   Write a parser at pkg/parser/<toolname>/parser.go that follows the same
   patterns.
   ```

4. **Wire it up.** Add a `case "<toolname>"` in `cmd/report/report.go` and
   write a README for the new parser (see `pkg/executor/zizmor/README.md` for
   an example).

## Existing parsers

| Package | Tool | Input format |
|---------|------|-------------|
| `gotest` | `go test -json` | JSON (one object per line) |
| `gobench` | `go test -bench -json` | JSON (one object per line) |
| `k6` | k6 load tests | JSON summary |
| `playwright` | Playwright test runner | JSON report |

The zizmor parser lives in `pkg/executor/zizmor/` for now (bench does not run
zizmor but the package predates this separation). It will move here in a future
cleanup.

## Metric naming conventions

- No `bench_` prefix on parser-specific metrics. The Prometheus reporter
  already applies a `job="bench"` label, which is sufficient to filter by
  bench in Grafana.
- Use `_completed = 1` metrics (e.g. `zizmor_scan_completed`) so you can
  detect repos that were never scanned — a missing metric is different from a
  zero count.
- Add a `repo` label where meaningful. The `cmd/report` command injects the
  `--suite-name` value as a `repo` label on all parser-generated metrics.
