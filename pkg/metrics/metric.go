package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidMetricFormat  = fmt.Errorf("invalid metric format")
	ErrInvalidHeadersFormat = fmt.Errorf("invalid headers format")
	ErrInvalidMetricsFile   = fmt.Errorf("invalid metrics file")
	ErrAccessingFile        = fmt.Errorf("error accessing file")

	reMetric       = regexp.MustCompile(`^(?P<name>\w+)(\{(?P<labels>[^}]*)\})?(=|\s+)(?P<value>\d+(\.\d+)?(e[+|-]\d+)?)((\s+)(?P<timestamp>\d+))?`)
	nameIndex      = reMetric.SubexpIndex("name")
	labelsIndex    = reMetric.SubexpIndex("labels")
	valueIndex     = reMetric.SubexpIndex("value")
	timestampIndex = reMetric.SubexpIndex("timestamp")

	//FIXME: this reges is capturing the comma at the end of the header
	reHeadersLine = regexp.MustCompile(`^(\w+(\{[^}]*\})?)((?:,)(\w+(\{[^}]*\})?))*$`)
	reHeader      = regexp.MustCompile(`(\w+(\{[^}]*\})?)`)
)

// Metric represents a single metric with a name, value and optional labels
type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp int64
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
func ParseMetric(metricStr string) (Metric, error) {
	matches := reMetric.FindStringSubmatch(metricStr)
	if matches == nil {
		return Metric{}, fmt.Errorf("%w %q", ErrInvalidMetricFormat, metricStr)
	}

	name := matches[nameIndex]
	labelsStr := matches[labelsIndex]
	valueStr := matches[valueIndex]
	timestampStr := matches[timestampIndex]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return Metric{}, fmt.Errorf("%w invalid value %q", ErrInvalidMetricFormat, valueStr)
	}

	timestamp, _ := strconv.Atoi(timestampStr)

	labels := make(map[string]string)
	if labelsStr != "" {
		labelPairs := strings.Split(labelsStr, ",")
		for _, pair := range labelPairs {
			k, v, sep := strings.Cut(pair, "=")
			if !sep || k == "" || v == "" {
				return Metric{}, fmt.Errorf("%w invalid label %q", ErrInvalidMetricFormat, labelsStr)
			}
			labels[strings.Trim(k, " ")] = strings.Trim(v, " ")
		}
	}

	return Metric{
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: int64(timestamp),
	}, nil
}

// ParseHeader parses a string containing a list of headers in the format
// "name{label1=value1,label2=value2,...},name{label1=value1,label2=value2,...},..."
func ParseHeaders(headersLine string) ([]string, error) {
	if !reHeadersLine.MatchString(headersLine) {
		return nil, fmt.Errorf("%w %q", ErrInvalidHeadersFormat, headersLine)
	}

	headerList := reHeader.FindAllString(headersLine, -1)
	if headerList == nil {
		return nil, fmt.Errorf("%w %q", ErrInvalidHeadersFormat, headersLine)
	}

	// TODO: remove this when the regex is fixed
	cleanHeaders := make([]string, 0, len(headerList))
	for _, header := range headerList {
		cleanHeaders = append(cleanHeaders, strings.Trim(header, ","))
	}
	return cleanHeaders, nil
}

func ParseMetricsFile(path string) ([]Metric, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrAccessingFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	metrics := []Metric{}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}

		metric, err := ParseMetric(line)
		if err != nil {
			return nil, fmt.Errorf("%w %w", ErrInvalidMetricsFile, err)
		}
		metrics = append(metrics, metric)

	}
	return metrics, nil
}

func ParseMetricsCSVFile(path string) ([]Metric, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrAccessingFile, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	headersLine, _, err := reader.ReadLine()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w %w", ErrAccessingFile, err)
	}

	headers, err := ParseHeaders(string(headersLine))
	if err != nil {
		return nil, fmt.Errorf("%w %w", ErrInvalidMetricsFile, err)
	}

	valuesLine, _, err := reader.ReadLine()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w %w", ErrInvalidMetricsFile, err)
	}
	values := strings.Split(string(valuesLine), ",")
	if len(headers) != len(values) {
		return nil, fmt.Errorf("%w mismatched headers and values", ErrInvalidMetricsFile)
	}

	metrics := make([]Metric, 0, len(headers))
	for i, header := range headers {
		metric, err := ParseMetric(header + "=" + values[i])
		if err != nil {
			return nil, fmt.Errorf("%w %w", ErrInvalidMetricsFile, err)
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}
