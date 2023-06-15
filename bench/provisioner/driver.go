package provisioner

import "context"

// ProvisionDriver represents a provision driver interface used to provision an
// instance of Grafana and k6 test runner.
type ProvisionDriver interface {
	// Provision provisions the required resources.
	Provision(ctx context.Context, ps *ProvisionState) (func() error, error)

	// Blocking operation that waits for ProvisionState.GrafanaAddress to become responsive
	WaitForReady(ctx context.Context, ps *ProvisionState)

	// Checks to see if Grafana server is running on ProvisionState.GrafanaAddress
	Ready(ctx context.Context, ps *ProvisionState) bool

	// Destroy tears down the provisioned resources.
	Destroy(ctx context.Context, ps *ProvisionState) error
}
