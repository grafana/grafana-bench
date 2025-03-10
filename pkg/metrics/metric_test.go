package metrics

import (
	"errors"
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
