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

	// Directory containing state and work directories
	LocalDir string
	// Directory where the provisioner will store everything needed to boot
	// Grafana
	WorkDir string
	// Directory containing state information
	StateDir string

	// Grafana build the provision is based on
	Build *builder.Build

	// Temporary. should be refactored to live somewhere else probably
	GrafanaRevision     string
	GrafanaArtifactName string
	GrafanaPath         string

	// Custom setup info
	TemplateDir          string
	CustomGrafanaINIPath string

	// Results
	GrafanaAddress string
	K6Address      string
	killFunc       func() error
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
