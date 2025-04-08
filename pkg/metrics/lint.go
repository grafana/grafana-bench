package metrics

import (
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus/testutil/promlint/validations"
	dto "github.com/prometheus/client_model/go"
)

// LintMetrics takes in a list of metrics and converts them to metric families
// the core problem with this approach is that some of the promlint validations rely on
// MetricFamily.Type
func LintMetrics(log *slog.Logger, metrics []Metric, strictLint bool) error {
	for _, metric := range metrics {
		mf := &dto.MetricFamily{
			Name: &metric.Name,
		}

		anyFailures := false
		for _, fn := range prometheusValidations {
			errs := fn(mf)
			for _, err := range errs {
				anyFailures = true
				log.Warn("prometheus linter error", "metricName", metric.Name, "lintError", err.Error())
			}
		}

		if anyFailures && strictLint {
			return fmt.Errorf("prometheus linter too many errors")
		}
	}

	return nil
}

type Validation = func(mf *dto.MetricFamily) []error

// prometheusValidations is a list of linting validations to run against metrics
// we currently only run validations that ONLY rely on MetricFamily.Name
// you can find the full list of linters at
// https://github.com/prometheus/client_golang/blob/main/prometheus/testutil/promlint/validation.go#L24
var prometheusValidations = []Validation{
	validations.LintMetricUnits,
	validations.LintReservedChars,
	validations.LintCamelCase,
	validations.LintUnitAbbreviations,

	// validations.LintHistogramSummaryReserved,
	// validations.LintHelp, // not sure if we care about help
	// validations.LintMetricTypeInName, // relies on type
	// validations.LintCounter, // relies on type
	// validations.LintDuplicateMetric, // not entirely sure how this works
}
