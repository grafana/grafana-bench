package bench

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
)

func (b *Config) Run() error {
	var err error

	if err := b.ResolveConfig(); err != nil {
		return err
	}

	if err := b.Build(); err != nil {
		return err
	}

	fmt.Println("setting up work directory")
	executable, err := setupWorkdir(b)
	if err != nil {
		return err
	}

	fmt.Println("booting grafana")
	cmd := exec.Command(executable, "server")
	err = utils.DoInDir(b.ProjectRoot, "work", func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Wait for the server to start up
	for {
		_, err := net.Dial("tcp", "localhost:3000")
		if err == nil {
			fmt.Println("Server is ready!")
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(time.Second)
	}

	// wait for signal to kill grafana
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("Shutting down grafana process")
	return nil
}
