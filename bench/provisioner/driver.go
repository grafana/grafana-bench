package provisioner

import "context"

// ProvisionDriver represents a provision driver interface used to provision an
// instance of Grafana and k6 test runner.
type ProvisionDriver interface {
	// Provision provisions the required resources.
	Provision(ctx context.Context, ps *ProvisionState) error

	// Check performs a health check on the provisioned resources.
	Ready(ctx context.Context, ps *ProvisionState) error

	// Destroy tears down the provisioned resources.
	Destroy (ctx context.Context, ps *ProvisionState) error
}
