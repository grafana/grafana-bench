package analyze

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/grafana/grafana-bench/pkg/analyzer"
	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/utils/env"
	"github.com/spf13/cobra"
)

const longDescription = `
analyze subcommand queries Loki for msg=testRun events and emits msg=defectConfirmed
events for (service, runStage, testFile, grafanaVersion) tuples that breach the
defect-confirmation rule.

Rule (v2):
  1. Persistence:       ≥ --analyze-min-failures consecutive failing testRun
                        events on the target grafanaVersion.
  2. Regression boundary: the most recent prior grafanaVersion in the same
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
`

const examples = `
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
`

// NewCmd creates the analyze subcommand.
func NewCmd(log *slog.Logger) *cobra.Command {
	cfg := &config.AnalyzeConfig{}

	cmd := &cobra.Command{
		Use:     "analyze",
		Short:   "analyze testRun events in Loki and emit defectConfirmed events",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyEnvFallbacks(cfg)
			if err := validateConfig(cfg); err != nil {
				return err
			}

			lokiClient := analyzer.NewLokiClient(analyzer.LokiConfig{
				BaseURL:     cfg.LokiURL,
				Username:    cfg.LokiUsername,
				Password:    cfg.LokiPassword,
				BearerToken: cfg.LokiBearerToken,
			}, log)

			end := time.Now()
			start := end.Add(-cfg.Window)

			selector := buildSelector(cfg)
			log.Debug("querying loki for testRun events",
				"selector", selector,
				"start", start.Format(time.RFC3339),
				"end", end.Format(time.RFC3339))

			records, err := lokiClient.QueryTestRuns(cmd.Context(), selector, start, end)
			if err != nil {
				return fmt.Errorf("querying loki: %w", err)
			}

			filtered := filterRecords(records, cfg)
			log.Debug("loaded testRun records",
				"pulled", len(records),
				"after_filter", len(filtered))

			defects := analyzer.Analyze(filtered, analyzer.RuleConfig{
				MinFailures:     cfg.MinFailures,
				MinPriorPassing: cfg.MinPriorPassing,
			}, cfg.Window, time.Now())

			log.Debug("analysis complete", "defects", len(defects))

			if !cfg.Emit {
				for _, d := range defects {
					log.Info("dry-run defect",
						"testFile", d.TestFile,
						"grafanaVersion", d.GrafanaVersion,
						"priorPassingVersion", d.PriorPassingVersion,
						"confidence", d.Confidence,
						"retryExhausted", d.RetryExhausted,
						"signatureHash", d.SignatureHash,
						"confidenceRuns", d.ConfidenceRuns,
					)
				}
				return nil
			}

			return emit(defects, cfg, cmd.OutOrStdout())
		},
	}

	config.AddAnalyzeFlags(cmd.Flags(), cfg)
	return cmd
}

func applyEnvFallbacks(cfg *config.AnalyzeConfig) {
	if cfg.LokiURL == "" {
		cfg.LokiURL = env.EnvOrDefault("BENCH_LOKI_URL", "")
	}
	if cfg.LokiUsername == "" {
		cfg.LokiUsername = env.EnvOrDefault("BENCH_LOKI_USERNAME", "")
	}
	if cfg.LokiPassword == "" {
		cfg.LokiPassword = env.EnvOrDefault("BENCH_LOKI_PASSWORD", "")
	}
	if cfg.LokiBearerToken == "" {
		cfg.LokiBearerToken = env.EnvOrDefault("BENCH_LOKI_BEARER_TOKEN", "")
	}
}

func validateConfig(cfg *config.AnalyzeConfig) error {
	if cfg.LokiURL == "" {
		return fmt.Errorf("--analyze-loki-url or BENCH_LOKI_URL is required")
	}
	if cfg.Service == "" {
		return fmt.Errorf("--analyze-service is required")
	}
	if cfg.Window <= 0 {
		return fmt.Errorf("--analyze-window must be positive")
	}
	return nil
}

// buildSelector returns the LogQL query. The --analyze-loki-selector flag
// provides the stream-level filter; we do post-filtering on the decoded
// records rather than baking every analyzer knob into LogQL.
func buildSelector(cfg *config.AnalyzeConfig) string {
	return cfg.LokiSelector
}

func filterRecords(records []analyzer.TestRunRecord, cfg *config.AnalyzeConfig) []analyzer.TestRunRecord {
	out := make([]analyzer.TestRunRecord, 0, len(records))
	for _, r := range records {
		if r.Service != cfg.Service {
			continue
		}
		if cfg.RunStage != "" && r.RunStage != cfg.RunStage {
			continue
		}
		out = append(out, r)
	}
	return out
}

func emit(defects []analyzer.ConfirmedDefect, cfg *config.AnalyzeConfig, w io.Writer) error {
	emitter, err := analyzer.NewEmitter(w, cfg.Output, []any{"tool", "bench", "service", cfg.Service})
	if err != nil {
		return fmt.Errorf("building emitter: %w", err)
	}
	for _, d := range defects {
		emitter.EmitDefectConfirmed(d)
	}
	return nil
}

