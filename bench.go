package main

import (
	"context"
	"os"
	"log/slog"

	"github.com/grafana/grafana-bench/cmd/root"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	root := root.NewCmd(log)

	err := root.ExecuteContext(context.Background())
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
