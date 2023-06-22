package provisioner

import (
	"context"

	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/tester"
)

type ProvisionState struct {
	// UUID for the build
	Identifier string
	driver     ProvisionDriver

	// Identifies what type of provision is used, vm, local, or hosted grafana
	Type ProvisionType

	// Directory where the provisioner will store everything needed to provision
	// and boot a Grafana server
	LocalDir string
	// Directory containing files to boot grafana executable
	WorkDir string
	// Directory containing state information
	StateDir string

	// Grafana build the provision is based on
	Build *builder.Build

	// Custom setup info
	TemplateDir          string
	CustomGrafanaINIPath string

	// Grafana instance create on provision
	GrafanaInstance *VMInstance

	// K6 instance, only created when using a non-local driver
	K6VM     *VMInstance
	killFunc func() error
}

// Returns a function to shut down grafana. Does not destroy the infrastructure
// that was provisioned
func (p *ProvisionState) Provision(ctx context.Context) (func() error, error) {
	var err error
	p.killFunc, err = p.driver.Provision(ctx, p)
	return p.killFunc, err
}

func (p *ProvisionState) WaitForReady(ctx context.Context) {
	p.driver.WaitForReady(ctx, p)
}

func (p *ProvisionState) Destroy(ctx context.Context) error {
	return p.driver.Destroy(ctx, p)
}

func (p *ProvisionState) RunTests(ctx context.Context, tr *tester.TestRun) error {
	return p.driver.RunTests(ctx, p, tr)
}
