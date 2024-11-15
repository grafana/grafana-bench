package notifier

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const validYamlMapping = `
mapping:
  - slack_channel: slack-channel
    github_team: "@github-team"
  - slack_channel: null
    github_team: "@another-team"
`

func checkExpected(t *testing.T, mapping CodeownersMapping, expected map[string]string) {
	for recipient, expectedAddress := range expected {
		address, err := mapping.GetChannel(recipient)
		if address != expectedAddress {
			t.Fatalf("expected %s got %s", expectedAddress, address)
		}

		if address == "" && err != ErrNoMappingForCodeowner {
			t.Fatalf("expected address not found error got %v", err)
		}
	}
}

func TestYamMappingFromReader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title       string
		source      string
		expected    map[string]string
		expectedErr error
	}{
		{
			title:  "valid yaml",
			source: validYamlMapping,
			expected: map[string]string{
				"@github-team":  "slack-channel",
				"@another-team": "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			mapping, err := MappingFromReader(bytes.NewReader([]byte(tc.source)))
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			if tc.expectedErr != nil {
				return
			}

			checkExpected(t, mapping, tc.expected)
		})
	}
}

func TestMappingFromFile(t *testing.T) {
	t.Parallel()

	// create valid test yaml
	yamlFile := filepath.Join(t.TempDir(), "test.yaml")
	err := os.WriteFile(yamlFile, []byte(validYamlMapping), 0o644)
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	testCases := []struct {
		title       string
		path        string
		expectedErr error
	}{
		{
			title:       "valid path to yaml",
			path:        yamlFile,
			expectedErr: nil,
		},
		{
			title:       "yaml file doesn't exist",
			path:        "non-existing-file.yaml",
			expectedErr: ErrGettingMapping,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			mapping, err := MappingFromFile(tc.path)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			if tc.expectedErr != nil {
				return
			}

			// expected mapping between github team and slack channel
			expected := map[string]string{
				"@github-team":  "slack-channel",
				"@another-team": "",
			}

			checkExpected(t, mapping, expected)
		})
	}
}

func TestYamlDirectoryFromURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title       string
		handler     http.HandlerFunc
		expectedErr error
	}{
		{
			title: "valid url to yaml",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(validYamlMapping))
			},
			expectedErr: nil,
		},
		{
			title: "valid url to yaml",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedErr: ErrGettingMapping,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tc.handler)

			mapping, err := MappingFromULR(srv.URL)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v got %v", tc.expectedErr, err)
			}

			if tc.expectedErr != nil {
				return
			}

			// expected mapping between github team and slack channel
			expected := map[string]string{
				"@github-team":  "slack-channel",
				"@another-team": "",
			}

			checkExpected(t, mapping, expected)
		})
	}
}
