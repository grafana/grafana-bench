# pkg/parser

This package contains parsers for tool output formats. Each sub-package reads
the output of a specific tool and converts it into a `SuiteRunSummary` that
bench can report, log, or push as Prometheus metrics.

All parsers should live here. Create an executor if you need bench to execute
your tool (e.g. data is parsed from stdout or test portability).

## Writing a new parser

The recommended workflow:

1. **Collect 2–3 real output files** from the tool — ideally one with
   findings/failures, one clean, and one edge case (empty output, parse error,
   etc.). Drop them in `pkg/parser/<toolname>/testdata/`.

2. **Decide which metrics matter.** Write down the questions you would like to
   answer, then have an LLM generate those metrics in
   [Prometheus text exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/).

   For example, given a frontend coverage tool, you might ask:
   - Which teams have the lowest coverage?
   - Is coverage improving over time?
   - Which packages are below threshold?

   Keep the following conventions in mind when naming metrics:
   - No `bench_` prefix — the Prometheus reporter already applies a `job="bench"`
     label, which is sufficient to filter by bench in Grafana.
   - Use a `_completed = 1` metric so you can detect repos that were never
     scanned — a missing metric is different from a zero count.
   - Add a `repo` label where meaningful. The `cmd/report` command injects the
     `--suite-name` value as a `repo` label on all parser-generated metrics.

   The LLM should produce something like:

   ```
   # HELP mytool_line_coverage_percent Line test coverage percentage
   # TYPE mytool_line_coverage_percent gauge
   mytool_line_coverage_percent{codeowner="@org/team-a",package="@org/app"} 67.28 1769464898033
   mytool_line_coverage_percent{codeowner="@org/team-b",package="@org/app"} 40.59 1769464879545

   # HELP mytool_branch_coverage_percent Branch test coverage percentage
   # TYPE mytool_branch_coverage_percent gauge
   mytool_branch_coverage_percent{codeowner="@org/team-a",package="@org/app"} 55.05 1769464898033
   mytool_branch_coverage_percent{codeowner="@org/team-b",package="@org/app"} 41.85 1769464879545

   # HELP mytool_scan_completed Set to 1 when a scan completes (absence means repo was never scanned)
   # TYPE mytool_scan_completed gauge
   mytool_scan_completed{repo="org/repo"} 1 1769464898033
   ```

   Writing it in exposition format forces you to think through metric names and
   label cardinality upfront — and gives the LLM in step 3 a precise target.

3. **Point an LLM at the tool output + your metrics + this codebase.**
   A prompt like the following works well:

   ```
   Write me a new bench parser for <tool>.
   There are 2 sample outputs in pkg/parser/<tool>/testdata/.
   I want to generate metrics to answer these questions:
   - Which teams have the lowest coverage?
   - Is coverage improving over time?
   - Which packages are below threshold?

   The metrics should look roughly like:

   # HELP mytool_line_coverage_percent Line test coverage percentage
   # TYPE mytool_line_coverage_percent gauge
   mytool_line_coverage_percent{codeowner="@org/team-a",package="@org/app"} 67.28 1769464898033
   mytool_line_coverage_percent{codeowner="@org/team-b",package="@org/app"} 40.59 1769464879545

   # HELP mytool_branch_coverage_percent Branch test coverage percentage
   # TYPE mytool_branch_coverage_percent gauge
   mytool_branch_coverage_percent{codeowner="@org/team-a",package="@org/app"} 55.05 1769464898033
   mytool_branch_coverage_percent{codeowner="@org/team-b",package="@org/app"} 41.85 1769464879545

   # HELP mytool_scan_completed Set to 1 when a scan completes (absence means repo was never scanned)
   # TYPE mytool_scan_completed gauge
   mytool_scan_completed{repo="org/repo"} 1 1769464898033

   Here is an existing parser for reference: pkg/executor/zizmor/parser.go
   Here is the SuiteRunSummary type: pkg/executor/executor.go
   Write a parser at pkg/parser/<toolname>/parser.go that follows the same
   patterns and wire it up.

   Acceptance criteria:
   * parser has tests
   * `make test` passes
   * `make build` succeeds
   * run bench command against the test data and get expected output
   * there's a README for the parser explaining what it is and how it works
   * `make docs`
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
| `jscoverage` | JavaScript coverage (NYC/Istanbul, Jest, Vitest, Monocart, Playwright) | JSON summary |

The zizmor parser lives in `pkg/executor/zizmor/` for now (bench does not run
zizmor but the package predates this separation). It will move here in a future
cleanup.
