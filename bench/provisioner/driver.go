package provisioner

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/grafana/grafana-bench/bench/tester"
)

// ProvisionDriver represents a provision driver interface used to provision an
// instance of Grafana and k6 test runner.
type ProvisionDriver interface {
	// Provision provisions the required resources.
	Provision(ctx context.Context, ps *ProvisionState) (func() error, error)

	// Blocking operation that waits for ProvisionState.GrafanaAddress to become responsive
	WaitForReady(ctx context.Context, ps *ProvisionState)

	// Uses driver to run the test suite against instance of Grafana
	RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error

	// Gets the machine spec to pass in when running tests
	GetMachineSpec(ctx context.Context, ps *ProvisionState) (string, error)

	// Destroy tears down the provisioned resources.
	Destroy(ctx context.Context, ps *ProvisionState) error
}

// Stubbed function to return when something goes wrong provisioning
func NilFunc() error {
	return nil
}

// Wait for the server to start up
func WaitForLiveGrafana(address string) {
	for {
		if IsLive(address) {
			fmt.Println("Server is ready!")
			break
		}
		fmt.Printf("Waiting for server on %s...\n", address)
		time.Sleep(time.Second)
	}
}

func IsLive(address string) bool {
	_, err := net.Dial("tcp", address)
	return err == nil
}
