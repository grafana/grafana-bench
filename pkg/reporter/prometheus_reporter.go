package reporter

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/metrics"
	"github.com/grafana/grafana-bench/pkg/prometheus"

	prompb "buf.build/gen/go/prometheus/prometheus/protocolbuffers/go"
)

type PrometheusConfig struct {
	URL      string
	User     string
	Password string
	Timeout  time.Duration
	Prefix   string
}

type PrometheusReporter struct {
	name   string
	client *prometheus.Client
	prefix string
}

func (r PrometheusReporter) Name() string {
	return r.name
}

func NewPrometheusReporter(config PrometheusConfig) *PrometheusReporter {
	return &PrometheusReporter{
		name:   "prometheusReporter",
		prefix: config.Prefix,
		client: prometheus.New(
			config.URL,
			prometheus.Options{
				User:     config.User,
				Password: config.Password,
				Timeout:  config.Timeout,
			}),
	}
}

func makeTimeSeries(labels map[string]string, name string, timestamp time.Time, value float64) *prompb.TimeSeries {
	tsLabels := []*prompb.Label{
		{
			Name:  "__name__",
			Value: name,
		},
	}

	for label, value := range labels {
		tsLabels = append(tsLabels, &prompb.Label{Name: label, Value: value})
	}

	return &prompb.TimeSeries{
		Labels: tsLabels,
		Samples: []*prompb.Sample{
			{
				Value:     value,
				Timestamp: timestamp.UnixMilli(),
			},
		},
	}
}

func (p *PrometheusReporter) Report(
	ctx context.Context,
	suiteRun executor.SuiteRun,
	summary executor.SuiteRunSummary,
) error {
	var ts []*prompb.TimeSeries

	labels := map[string]string{
		"job":             "bench",
		"grafana_version": suiteRun.GrafanaVersion,
		"status":          string(summary.Status),
		"suite_run":       suiteRun.Name,
	}

	for k, v := range suiteRun.Attributes {
		labels[k] = v
	}

	reportMetrics := []metrics.Metric{
		{Name: "bench_tests_executed", Value: float64(summary.TestsExecuted), Labels: labels},
		{Name: "bench_tests_passed", Value: float64(summary.TestsPassed), Labels: labels},
		{Name: "bench_tests_failed", Value: float64(summary.TestsFailed), Labels: labels},
		{Name: "bench_tests_error", Value: float64(summary.TestsError), Labels: labels},
		{Name: "bench_tests_flaky", Value: float64(summary.TestsFlaky), Labels: labels},
		{Name: "bench_total_duration_seconds", Value: float64(summary.TotalDuration / 1000.0), Labels: labels},
	}

	// add prefix to custom metrics
	prefix := ""
	if p.prefix != "" {
		prefix = fmt.Sprintf("%s_", p.prefix)
	}

	for _, m := range summary.Metrics {
		metric := metrics.Metric{
			Name:      strings.ReplaceAll(fmt.Sprintf("%s%s", prefix, m.Name), "-", "_"),
			Value:     m.Value,
			Timestamp: m.Timestamp,
			Labels:    make(map[string]string),
		}
		maps.Copy(metric.Labels, labels)
		maps.Copy(metric.Labels, m.Labels)
		reportMetrics = append(reportMetrics, metric)
	}

	for _, metric := range reportMetrics {
		ts = append(
			ts,
			makeTimeSeries(
				metric.Labels,
				metric.Name,
				summary.StartTime,
				metric.Value,
			),
		)
	}

	return p.client.Push(ctx, ts)
}
