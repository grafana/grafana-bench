package reporter

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
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

func toPrometheusLabel(prefix string, label string) string {
	if prefix != "" {
		prefix = fmt.Sprintf("%s_", prefix)
	}
	return strings.ReplaceAll(fmt.Sprintf("%s%s", prefix, label), "-", "_",)
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
	}

	metrics := map[string]float64{
		"tests_executed": float64(summary.TestsExecuted),
		"tests_passed":   float64(summary.TestsPassed),
		"tests_failed":   float64(summary.TestsFailed),
		"tests_error":    float64(summary.TestsError),
		"tests_flaky":    float64(summary.TestsPassed),
	}

	if summary.Metrics != nil {
		maps.Copy(metrics, summary.Metrics)
	}

	prefix := fmt.Sprintf("%s%s", p.prefix, suiteRun.Name)

	for metric, value := range metrics {
		ts = append(
			ts,
			makeTimeSeries(
				labels,
				toPrometheusLabel(prefix, metric),
				summary.StartTime,
				value,
			),
		)
	}

	return p.client.Push(ctx, ts)
}
