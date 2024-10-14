package openapi

import (
	"errors"
	"testing"
)

func TestParsing(t *testing.T) {
	testCase := []struct{
		title     string
		document  string
		expectErr error
		expect    map[string][]string
	}{
		{
			title:    "parse grafana api v3",
			document: "testdata/grafanaV3.json",
			expectErr: nil,
			expect: map[string][]string{
				"/datasources": []string{"get", "post"},
				"/datasources/name/{name}": []string{"get", "delete"},
			},
		},
	}

	for _, tc := range testCase {
		api, err := FromFile(tc.document)
		if !errors.Is(err, tc.expectErr) {
			t.Fatalf("expected error %v got %v", tc.expectErr, err)
		}

		if tc.expectErr != nil {
			return
		}

		t.Run("test operations", func(t *testing.T) {
			for path, expected := range tc.expect {
				actual, err := api.GetOperations(path)
				if err != nil {
					t.Fatalf("expected path %s not found", path)
				}

				if len(actual) != len(expected) {
					t.Fatalf("expected %d operations for path %s, got %d", len(expected), path, len(actual))
				}

				for _, op := range expected {
					if actual[op] == "" {
						t.Fatalf("operation %s not found for path %s", op, path)
					}
				}
			}
		})
	}
}
