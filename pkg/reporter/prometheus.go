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
	client *prometheus.Client
	prefix string
}

func NewPrometheusReporter(config PrometheusConfig) *PrometheusReporter {
	return &PrometheusReporter{
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
		"grafana_version": suiteRun.GrafanaVersion,
		"status":          string(summary.Status),
		"suite_run":       suiteRun.Name,
	}

	metrics := []metrics.Metric{
		{Name: "tests_executed", Value: float64(summary.TestsExecuted), Labels: labels},
		{Name: "tests_passed", Value: float64(summary.TestsPassed), Labels: labels},
		{Name: "tests_failed", Value: float64(summary.TestsFailed), Labels: labels},
		{Name: "tests_error", Value: float64(summary.TestsError), Labels: labels},
		{Name: "tests_flaky", Value: float64(summary.TestsPassed), Labels: labels},
		{Name: "total_duration_seconds", Value: float64(summary.TotalDuration / 1000.0), Labels: labels},
	}

	for _, m := range summary.Metrics {
		if m.Labels == nil {
			m.Labels = make(map[string]string)
		}
		maps.Copy(m.Labels, labels)
		metrics = append(metrics, m)
	}

	prefix := "bench_suite_run" 
	if p.prefix != "" {
		prefix = fmt.Sprintf("%s_%s", prefix, p.prefix)	
	}

	for _, metric := range metrics {
		name := fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(metric.Name, "-", "_"))
		ts = append(
			ts,
			makeTimeSeries(
				metric.Labels,
				name,
				summary.StartTime,
				metric.Value,
			),
		)
	}

	return p.client.Push(ctx, ts)
}
