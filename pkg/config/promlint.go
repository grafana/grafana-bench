package config

import (
	"fmt"
	"os"

	"github.com/grafana/grafana-bench/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint/validations"
	dto "github.com/prometheus/client_model/go"
)

// LintFile the metrics file using promlint
// promlint only has support for linting a text exposition format which is why we read the file again.
// It's possible to write some wrappers to lint names, but we very quickly run into
// cases where we need to create a "metric family" which relies on internal structures or have to hand
// pick linters that only rely on name
func LintFile(metricFile string, strictLint bool) error {
	f, err := os.Open(metricFile)
	linter := promlint.New(f)
	problems, err := linter.Lint()
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		for _, v := range problems {
			fmt.Printf("\n WARN: prometheus linter - %s - %s", v.Metric, v.Text)
		}

		if strictLint {
			return fmt.Errorf("prometheus linter too many errors")
		}
	}

	return nil
}

// LintMetrics takes in a list of metrics and converts them to metric families
// the core problem with this approach is that some of the promlint validations rely on
// MetricFamily.Type
func LintMetrics(metrics []metrics.Metric, strictLint bool) error {

	for _, metric := range metrics {

		mf := &dto.MetricFamily{
			Name: &metric.Name,
		}

		anyFailures := false
		for _, fn := range defaultValidations {
			errs := fn(mf)
			for _, err := range errs {
				anyFailures = true
				fmt.Printf("\n WARN: prometheus linter - %s - %s", metric.Name, err.Error())
			}
		}

		if anyFailures && strictLint {
			return fmt.Errorf("prometheus linter too many errors")
		}
	}

	return nil
}

// Default validations to use from promlint
type Validation = func(mf *dto.MetricFamily) []error

var defaultValidations = []Validation{
	validations.LintMetricUnits,
	validations.LintReservedChars,
	validations.LintCamelCase,
	validations.LintUnitAbbreviations,

	// These linters rely on things  other than Name

	// validations.LintHistogramSummaryReserved,
	// validations.LintHelp, // not sure if we care about help
	// validations.LintMetricTypeInName, // relies on type
	// validations.LintCounter, // relies on type
	// validations.LintDuplicateMetric, // not entirely sure how this works
}
