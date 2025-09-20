package test

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestTestFailureErrorTypeCheck(t *testing.T) {
	err := TestFailureError{message: "test suite failed"}
	if err.Error() != "test suite failed" {
		t.Errorf("TestFailureError.Error() = %v, want %v", err.Error(), "test suite failed")
	}

	var target TestFailureError
	if !errors.As(err, &target) {
		t.Error("expected TestFailureError to be identified with errors.As")
	}
}

func TestMainExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "test-bench", "../../bench.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}
	defer func() { _ = os.Remove("test-bench") }()

	tests := []struct {
		name         string
		args         []string
		expectedExit int
	}{
		{
			name:         "invalid config should exit 2",
			args:         []string{"test", "--grafana-url", "invalid", "--test-suite", "nonexistent"},
			expectedExit: 2,
		},
		{
			name:         "help should exit 0",
			args:         []string{"test", "--help"},
			expectedExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./test-bench", tt.args...)
			err := cmd.Run()

			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("unexpected error type: %v", err)
				}
			}

			if exitCode != tt.expectedExit {
				t.Errorf("expected exit code %d, got %d", tt.expectedExit, exitCode)
			}
		})
	}
}
