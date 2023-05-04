package bench

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
)

func (b *Config) Boot(ctx context.Context, executable string) (func(), error) {
	cmd := exec.Command(executable, "server")

	// function to return so we can kill the process
	killFunc := func() {
		err := cmd.Process.Kill()
		if err != nil {
			fmt.Println("ERROR killing grafana PID:", err)
		}
	}

	err := utils.DoInDir(b.ProjectRoot, "work", func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})

	if err != nil {
		return killFunc, err
	}

	return killFunc, nil
}

// Wait for the server to start up
func waitForLiveGrafana() {
	for {
		_, err := net.Dial("tcp", "localhost:3000")
		if err == nil {
			fmt.Println("Server is ready!")
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(time.Second)
	}
}
