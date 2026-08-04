package analyzer

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Emitter writes msg=defectConfirmed events to an underlying slog logger.
// It mirrors the construction pattern used by pkg/reporter.LogReporter so
// that defectConfirmed events land in Loki in the same logfmt/JSON shape
// as every other bench event.
type Emitter struct {
	log *slog.Logger
}

// NewEmitter builds an Emitter writing to w using the given format ("json" or
// "text"). The baseAttrs are applied once (typically `tool=bench` and the
// service name) so every emitted event inherits them.
func NewEmitter(w io.Writer, format string, baseAttrs []any) (*Emitter, error) {
	var log *slog.Logger
	switch strings.ToLower(format) {
	case "json", "":
		log = slog.New(slog.NewJSONHandler(w, nil))
	case "text":
		log = slog.New(slog.NewTextHandler(w, nil))
	default:
		return nil, fmt.Errorf("unsupported emitter format: %s", format)
	}
	log = log.With(baseAttrs...)
	return &Emitter{log: log}, nil
}

// DefaultEmitter writes JSON events to stdout with tool=bench, service=svc.
func DefaultEmitter(svc string) (*Emitter, error) {
	return NewEmitter(os.Stdout, "json", []any{"tool", "bench", "service", svc})
}

// EmitDefectConfirmed writes a single msg=defectConfirmed event.
func (e *Emitter) EmitDefectConfirmed(d ConfirmedDefect) {
	attrs := []any{
		"runStage", d.RunStage,
		"testFile", d.TestFile,
		"testFolder", d.TestFolder,
		"grafanaVersion", d.GrafanaVersion,
		"priorPassingVersion", d.PriorPassingVersion,
		"grafanaSlug", d.GrafanaSlug,
		"grafanaUrl", d.GrafanaURL,
		"signatureHash", d.SignatureHash,
		"confidence", d.Confidence,
		"confidenceRuns", d.ConfidenceRuns,
		"priorPassingRuns", d.PriorPassingRuns,
		"retryExhausted", d.RetryExhausted,
		"exitMessageCanonical", d.ExitMessageCanonical,
		"exitMessageSample", d.ExitMessageSample,
		"firstFailureTime", d.FirstFailureTime.Format(time.RFC3339),
		"lastFailureTime", d.LastFailureTime.Format(time.RFC3339),
		"sourceRunIds", strings.Join(d.SourceRunIDs, ","),
		"analyzeWindowSeconds", int(d.AnalyzeWindow.Seconds()),
		"analyzedAt", d.AnalyzedAt.Format(time.RFC3339),
		"ruleVersion", d.RuleVersion,
	}
	// The second positional arg to Info mirrors LogReporter.Report's
	// "suiteRun", "anyFailures", … idiom — slog uses it as the first attribute.
	e.log.With(attrs...).Info("defectConfirmed", "defectConfirmed", d.SignatureHash)
}
