package metrics

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseMetric(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *Metric
		wantErr     error
		errContains string
	}{
		{
			name:  "basic metric without labels",
			input: "requests=42.0",
			want: &Metric{
				Name:   "requests",
				Value:  42.0,
				Labels: map[string]string{},
			},
			wantErr: nil,
		},
		{
			name:  "metric with integer value",
			input: "response_time=42",
			want: &Metric{
				Name:   "response_time",
				Value:  42,
				Labels: map[string]string{},
			},
			wantErr: nil,
		},
		{
			name:  "metric with single label",
			input: "requests{endpoint=/api/search}=10",
			want: &Metric{
				Name:   "requests",
				Value:  10,
				Labels: map[string]string{"endpoint": "/api/search"},
			},
			wantErr: nil,
		},
		{
			name:  "metric with label with spaces",
			input: "requests{ endpoint= /api/search }=10",
			want: &Metric{
				Name:   "requests",
				Value:  10,
				Labels: map[string]string{"endpoint": "/api/search"},
			},
			wantErr: nil,
		},
		{
			name:  "metric with multiple labels",
			input: "requests{endpoint=/api/search,method=GET,status=200}=25",
			want: &Metric{
				Name:  "requests",
				Value: 25,
				Labels: map[string]string{
					"endpoint": "/api/search",
					"method":   "GET",
					"status":   "200",
				},
			},
			wantErr: nil,
		},
		{
			name:        "invalid metric format - missing value",
			input:       "requests{endpoint=/api/search}",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid metric format",
		},
		{
			name:        "invalid metric format - missing name",
			input:       "{endpoint=/api/search}=10",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid metric format",
		},
		{
			name:        "invalid metric format - bad value",
			input:       "requests=abc",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid metric",
		},
		{
			name:        "invalid label format",
			input:       "requests{badlabel}=10",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid label",
		},
		{
			name:        "empty input",
			input:       "",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid metric",
		},
		{
			name:        "malformed label",
			input:       "requests{key1=value1,key2=}=10",
			want:        nil,
			wantErr:     ErrInvalidMetricFormat,
			errContains: "invalid label",
		},
		{
			name:  "zero value metric",
			input: "errors=0",
			want: &Metric{
				Name:   "errors",
				Value:  0,
				Labels: map[string]string{},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMetric(tt.input)

			// Check error condition
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("wantErr %v got %v", tt.wantErr, err)
				return
			}

			// If expecting error, check error message contains expected text
			if tt.wantErr != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %v should contain %s", err, tt.errContains)
					return
				}
			}

			// Check returned value
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %v got %v", tt.want, got)
			}
		})
	}
}
func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr error
	}{
		{
			name:    "single header without labels",
			input:   "requests",
			want:    []string{"requests"},
			wantErr: nil,
		},
		{
			name:    "single header with labels",
			input:   "requests{endpoint=/api/search,method=GET}",
			want:    []string{"requests{endpoint=/api/search,method=GET}"},
			wantErr: nil,
		},
		{
			name:    "multiple headers without labels",
			input:   "requests,errors",
			want:    []string{"requests", "errors"},
			wantErr: nil,
		},
		{
			name:    "multiple headers with labels",
			input:   "requests{endpoint=/api/search},errors{code=500}",
			want:    []string{"requests{endpoint=/api/search}", "errors{code=500}"},
			wantErr: nil,
		},
		{
			name:    "headers with mixed labels",
			input:   "requests{endpoint=/api/search},errors",
			want:    []string{"requests{endpoint=/api/search}", "errors"},
			wantErr: nil,
		},
		{
			name:    "invalid headers format",
			input:   "requests{endpoint=/api/search",
			want:    nil,
			wantErr: ErrInvalidHeadersFormat,
		},
		{
			name:    "empty input",
			input:   "",
			want:    nil,
			wantErr: ErrInvalidHeadersFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeaders(tt.input)

			// Check error condition
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("wantErr %v got %v", tt.wantErr, err)
				return
			}

			if tt.wantErr != nil {
				return
			}

			// Check returned value
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %v got %v", tt.want, got)
			}
		})
	}
}

func TestParseMetricsFile(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		want        []*Metric
		wantErr     error
		errContains string
	}{
		{
			name:     "valid metrics file",
			file: filepath.Join("testdata", "valid.csv"),
			want: []*Metric{
				{
					Name:  "requests",
					Value: 25,
					Labels: map[string]string{
						"endpoint": "/api/search",
					},
				},
				{
					Name:   "errors",
					Value:  1,
					Labels: map[string]string{},
				},
			},
			wantErr: nil,
		},
		{
			name:        "mismatched headers and values",
			file:       filepath.Join("testdata", "mismatched.csv"),
			want:        nil,
			wantErr:     ErrInvalidMetricsFile,
			errContains: "mismatched headers and values",
		},
		{
			name:        "missing headers",
			file:       filepath.Join("testdata", "noheaders.csv"),
			want:        nil,
			wantErr:     ErrAccessingFile,
		},
		{
			name:        "missing values",
			file:       filepath.Join("testdata", "novalues.csv"),
			want:        nil,
			wantErr:     ErrAccessingFile,
		},
		{
			name:        "empty file",
			file:       filepath.Join("testdata", "empty.csv"),
			want:        nil,
			wantErr:     ErrAccessingFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMetricsFile(tt.file)

			// Check error condition
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("wantErr %v got %v", tt.wantErr, err)
				return
			}

			// If expecting error, check error message contains expected text
			if tt.wantErr != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %v should contain %s", err, tt.errContains)
					return
				}
			}

			// Check returned value
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %v got %v", tt.want, got)
			}
		})
	}
}
