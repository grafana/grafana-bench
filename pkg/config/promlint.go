//package config

//import (
//  "github.com/grafana/grafana-bench/pkg/metrics"
//  "github.com/prometheus/client_golang/prometheus/testutil/promlint"
//  "github.com/prometheus/client_golang/prometheus/testutil/promlint/validations"
//  dto "github.com/prometheus/client_model/go"
//)

////type Validation = func(mf *dto.MetricFamily) []error

////var defaultValidations = []Validation{
////  validations.LintHelp,
////  validations.LintMetricUnits,
////  validations.LintCounter,
////  validations.LintHistogramSummaryReserved,
////  validations.LintMetricTypeInName,
////  validations.LintReservedChars,
////  validations.LintCamelCase,
////  validations.LintUnitAbbreviations,
////  validations.LintDuplicateMetric,
////}

////func (config *BenchConfig) LintMetrics(metrics []metrics.Metric) []promlint.Problem {
////  var problems []promlint.Problem

////  mfs := MetricsToMetricFamily(metrics)

////  for _, mf := range mfs {
////    for _, fn := range defaultValidations {
////      errs := fn(mf)
////      for _, err := range errs {
////        problems = append(problems, newProblem(mf, err.Error()))
////      }
////    }
////  }

////  return problems
////}

////// newProblem is helper function to create a Problem.
////func newProblem(mf *dto.MetricFamily, text string) promlint.Problem {
////  return promlint.Problem{
////    Metric: mf.GetName(),
////    Text:   text,
////  }
////}

////func MetricsToMetricFamily(metrics []metrics.Metric) []dto.MetricFamily {
////  mfs := []dto.MetricFamily{}

////  for _, m := range metrics {
////    mfs = append(mfs, dto.MetricFamily{
////      Name: &m.Name,
////    })

////  }

////  return mfs
////}
