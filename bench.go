package main

import (
	"context"
	"errors"
	"os"

	"github.com/grafana/grafana-bench/cmd/root"
	testcmd "github.com/grafana/grafana-bench/cmd/test"
	"github.com/grafana/grafana-bench/pkg/utils/logger"
)

func main() {
	log := logger.NewLogger("tool", "bench")

	root := root.NewCmd(log)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		log.Error(err.Error())

		// Handle custom exit codes for test command
		var testFailureErr testcmd.TestFailureError
		if errors.As(err, &testFailureErr) {
			os.Exit(1) // Test suite failed
		} else {
			os.Exit(2) // All other errors
		}
	}
}
