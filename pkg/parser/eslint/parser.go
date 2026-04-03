package eslint

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// Report represents the top-level ESLint suppressions report produced by
// `npm run lint:suppressions:by-codeowner` (see
// https://github.com/grafana/grafana/pull/119446).
type Report struct {
	Summary []Entry `json:"summary"`
}

// Entry represents a single codeowner's aggregated suppression count.
type Entry struct {
	Codeowner    *string  `json:"codeowner"`
	Suppressions *float64 `json:"suppressions"`
}

// ParseESLintReport reads an ESLint suppressions JSON report and returns a
// SuiteRunSummary. If codeowner is missing on any entry the parser returns an
// error. If suppressions is missing the entry is silently skipped.
//
// The pkg parameter maps to the --js-coverage-package CLI flag and is stored
// as a "package" attribute on the summary.
func ParseESLintReport(report io.Reader, pkg string) (executor.SuiteRunSummary, error) {
	var r Report
	if err := json.NewDecoder(report).Decode(&r); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("decoding ESLint suppressions JSON: %w", err)
	}

	now := time.Now()

	summary := executor.SuiteRunSummary{
		StartTime:  now,
		Status:     executor.SuitePassed,
		Attributes: map[string]string{},
	}

	if pkg != "" {
		summary.Attributes["package"] = pkg
	}

	for i, entry := range r.Summary {
		if entry.Codeowner == nil || *entry.Codeowner == "" {
			return executor.SuiteRunSummary{}, fmt.Errorf("entry %d is missing required field \"codeowner\"", i)
		}

		// suppressions is optional — skip the entry but do not error.
		if entry.Suppressions == nil {
			continue
		}
	}

	return summary, nil
}
