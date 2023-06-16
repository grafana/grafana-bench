package provisioner

import (
	"context"

	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/tester"
)

type ProvisionState struct {
	// UUID for the build
	Identifier string

	driver   ProvisionDriver
	Type     ProvisionType
	WorkDir  string
	StateDir string

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
}

// Returns a function to shut down grafana. Does not destroy the infrastructure
// that was provisioned
func (p *ProvisionState) Provision(ctx context.Context) (func() error, error) {
	return p.driver.Provision(ctx, p)
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
