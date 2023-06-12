package provisioner

import "context"

type ProvisionState struct {
	// UUID for the build
	Identifier string

	driver   ProvisionDriver
	Type     ProvisionType
	WorkDir  string
	StateDir string

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

func (p *ProvisionState) Provision(ctx context.Context) error {
	return p.driver.Provision(ctx, p)
}

func (p *ProvisionState) Check(ctx context.Context) error {
	return p.driver.Ready(ctx, p)
}

func (p *ProvisionState) Destroy(ctx context.Context) error {
	return p.driver.Destroy(ctx, p)
}
