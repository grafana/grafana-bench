package metrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidMetricFormat = fmt.Errorf("invalid metric format")

	re          = regexp.MustCompile(`^(?P<name>\w+)(\{(?P<labels>[^}]*)\})?=(?P<value>\d+(\.\d+)?)$`)
	nameIndex   = re.SubexpIndex("name")
	labelsIndex = re.SubexpIndex("labels")
	valueIndex  = re.SubexpIndex("value")
)

// Metric represents a single metric with a name, value and optional labels
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// parseMetric parses a metric string in the format "name{label1=value1,label2=value2,...}=value"
// and returns a Metric struct. The labels are optional
// Example: "requests{endpoint=/api/search,method=GET,status=200}=25"
// Returns:
//
//	 Metric{
//	     Name: "requests",
//	     Value: 25.0,
//	     Labels: map[string]string{"endpoint": "/api/search", "method": "GET", "status": "200"},
//	}
func ParseMetric(metricStr string) (*Metric, error) {
	matches := re.FindStringSubmatch(metricStr)
	if matches == nil {
		return nil, fmt.Errorf("%w %q", ErrInvalidMetricFormat, metricStr)
	}

	name := matches[nameIndex]
	labelsStr := matches[labelsIndex]
	valueStr := matches[valueIndex]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("%w invalid value %q", ErrInvalidMetricFormat, valueStr)
	}

	labels := make(map[string]string)
	if labelsStr != "" {
		labelPairs := strings.Split(labelsStr, ",")
		for _, pair := range labelPairs {
			k, v, sep := strings.Cut(pair, "=")
			if !sep || k == "" || v == "" {
				return nil, fmt.Errorf("%w invalid label %q", ErrInvalidMetricFormat, labelsStr)
			}
			labels[strings.Trim(k, " ")] = strings.Trim(v, " ")
		}
	}

	return &Metric{
		Name:   name,
		Value:  value,
		Labels: labels,
	}, nil
}
