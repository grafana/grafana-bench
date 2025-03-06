package reporter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)


func Test_ValidateLogStruct(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase  string
		logFile   string
		expectErr error
	}{
		{
			testCase:  "valid log file",
			logFile:   filepath.Join("testdata", "valid.log"),
			expectErr: nil,
		},
		{
			testCase:  "invalid time",
			logFile:   filepath.Join("testdata", "invalid_time.log"),
			expectErr: ErrInvalidLogFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			t.Parallel()

			logFile, err := os.Open(tc.logFile)
			if err != nil {
				t.Fatalf("failed to open log file %s: %s", tc.logFile, err)
			}
			defer logFile.Close()

			err = ValidateLog(logFile)
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("expected error %v, got %v", tc.expectErr, err)
			}
		})
	}
}