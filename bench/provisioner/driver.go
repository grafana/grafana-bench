package provisioner

import (
	"context"
	"log/slog"
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

type TestDriver interface {
	Setup(ctx context.Context)
	BuildCommand(ctx context.Context) error
	RunCommand(ctx context.Context) error
	PostProcess(ctx context.Context, fn []func() error) error
	Log(ctx context.Context) error
	Cleanup(ctx context.Context) error
}

// Stubbed function to return when something goes wrong provisioning
func NilFunc() error {
	return nil
}

// Wait for the server to start up
func WaitForLiveGrafana(log *slog.Logger, address string) {
	for {
		if IsLive(log, address) {
			log.Info("Grafana server is ready!")
			break
		}
		log.Info("Waiting for grafana server...", "address", address)
		time.Sleep(time.Second)
	}
}

func IsLive(log *slog.Logger, address string) bool {
	_, err := net.Dial("tcp", address)
	if err != nil {
		log.Info("Checking isLive...", "error", err)
	}
	return err == nil
}
