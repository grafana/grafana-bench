package config

import (
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func TestBenchConfig_GetRunAttributes(t *testing.T) {
	// Create a test logger that discards output to avoid cluttering test output
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name        string
		attributes  []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:       "empty attributes",
			attributes: []string{},
			expected:   map[string]string{},
		},
		{
			name:       "single attribute",
			attributes: []string{"key=value"},
			expected:   map[string]string{"key": "value"},
		},
		{
			name:       "multiple attributes in single string",
			attributes: []string{"key1=value1,key2=value2"},
			expected:   map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:       "multiple attribute strings",
			attributes: []string{"key1=value1", "key2=value2"},
			expected:   map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:       "whitespace handling",
			attributes: []string{" key1 = value1 , key2 = value2 "},
			expected:   map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:       "value with equals sign",
			attributes: []string{"url=http://example.com?param=value"},
			expected:   map[string]string{"url": "http://example.com?param=value"},
		},
		{
			name:       "overlapping keys - last wins",
			attributes: []string{"key=first", "key=second"},
			expected:   map[string]string{"key": "second"},
		},
		{
			name:       "overlapping keys in same string - last wins",
			attributes: []string{"key=first,key=second"},
			expected:   map[string]string{"key": "second"},
		},
		{
			name:       "empty values are skipped",
			attributes: []string{"key1=value1,key2=,key3=value3"},
			expected:   map[string]string{"key1": "value1", "key3": "value3"},
		},
		{
			name:       "empty attribute strings are skipped",
			attributes: []string{"", "key=value", "  "},
			expected:   map[string]string{"key": "value"},
		},
		{
			name:       "empty attributes in comma-separated string are skipped",
			attributes: []string{"key1=value1,,key2=value2"},
			expected:   map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:        "missing equals sign",
			attributes:  []string{"invalidkey"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty key",
			attributes:  []string{"=value"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "whitespace-only key",
			attributes:  []string{"  =value"},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BenchConfig{
				SuiteRun: SuiteRunConfig{
					Attributes: tt.attributes,
				},
			}

			result, err := config.GetRunAttributes(logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}