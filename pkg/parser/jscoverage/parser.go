package jscoverage

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/metrics"
)

type CoverageReport struct {
	Summary Summary `json:"summary"`
}

type Summary struct {
	Lines      *Coverage `json:"lines"`
	Statements *Coverage `json:"statements"`
	Functions  *Coverage `json:"functions"`
	Branches   *Coverage `json:"branches"`
}

type Coverage struct {
	Pct *float64 `json:"pct"`
}

func ParseCoverageJSON(report io.Reader, codeowner, pkg string) (executor.SuiteRunSummary, error) {
	var coverage CoverageReport
	decoder := json.NewDecoder(report)
	if err := decoder.Decode(&coverage); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("decoding JavaScript coverage JSON: %w", err)
	}

	summary := executor.SuiteRunSummary{
		StartTime: time.Now(),
		Status:    executor.SuitePassed,
		SuiteName: "jscoverage",
	}

	if summary.Attributes == nil {
		summary.Attributes = make(map[string]string)
	}
	summary.Attributes["codeowner"] = codeowner
	summary.Attributes["package"] = pkg

	timestamp := summary.StartTime.UnixMilli()
	labels := map[string]string{
		"codeowner": codeowner,
		"package":   pkg,
	}

	if coverage.Summary.Lines != nil && coverage.Summary.Lines.Pct != nil {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:      "js_coverage_lines_percent",
			Value:     *coverage.Summary.Lines.Pct,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	if coverage.Summary.Statements != nil && coverage.Summary.Statements.Pct != nil {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:      "js_coverage_statements_percent",
			Value:     *coverage.Summary.Statements.Pct,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	if coverage.Summary.Functions != nil && coverage.Summary.Functions.Pct != nil {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:      "js_coverage_functions_percent",
			Value:     *coverage.Summary.Functions.Pct,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	if coverage.Summary.Branches != nil && coverage.Summary.Branches.Pct != nil {
		summary.Metrics = append(summary.Metrics, metrics.Metric{
			Name:      "js_coverage_branches_percent",
			Value:     *coverage.Summary.Branches.Pct,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	summary.Metrics = append(summary.Metrics, metrics.Metric{
		Name:      "js_coverage_scan_completed",
		Value:     1.0,
		Labels:    copyLabels(labels),
		Timestamp: timestamp,
	})

	return summary, nil
}

func copyLabels(labels map[string]string) map[string]string {
	copied := make(map[string]string, len(labels))
	maps.Copy(copied, labels)
	return copied
}
