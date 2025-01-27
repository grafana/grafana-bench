package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-bench/cmd/root"
	"github.com/grafana/grafana-bench/pkg/utils/logger"
)

func main() {
	log := logger.NewLogger("service", "bench")

	root := root.NewCmd(log)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
