package bench

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (b *BenchRun) Run(ctx context.Context) error {
	var err error

	if err := b.ResolveConfig(ctx); err != nil {
		return err
	}

	if err := b.ResolveGrafanaBuild(ctx, true); err != nil {
		return err
	}

	fmt.Println("setup: copying files to working directory")
	executable, err := b.setupWorkdir(ctx)
	if err != nil {
		return err
	}

	// boot grafana
	fmt.Println("setup: booting grafana")
	killFunc, err := b.Boot(ctx, executable)
	if err != nil {
		return err
	}
	defer killFunc()

	waitForLiveGrafana()

	// wait for signal to kill grafana
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("Shutting down grafana process")
	return nil
}
