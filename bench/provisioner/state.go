package provisioner

import (
	"context"
	"encoding/json"
	"os"
	"path"

	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/tester"
)

type ProvisionState struct {
	// UUID for the build
	Identifier string `json:"identifier"`
	driver     ProvisionDriver

	// Identifies what type of provision is used, vm, local, or hosted grafana
	Type ProvisionType `json:"type"`

	// Directory where the provisioner will store state, bundle, and work
	// directory needed to provision and boot a Grafana server
	LocalDir string `json:"localDir"`
	// Directory containing files to boot grafana executable
	WorkDir string `json:"workDir"`
	// Directory containing state information
	StateDir string `json:"stateDir"`

	// Grafana build the provision is based on
	Build *builder.Build `json:"build"`

	// Custom setup info
	TemplateDir          string `json:"templateDir"`
	CustomGrafanaINIPath string `json:"customGrafanaINIPath"`

	// Grafana instance create on provision
	GrafanaInstance *VMInstance `json:"grafanaInstance"`

	// K6 instance, only created when using a non-local driver
	K6Instance *VMInstance `json:"k6Instance"`

	killFunc func() error
}

// Returns a function to shut down grafana. Does not destroy the infrastructure
// that was provisioned
func (p *ProvisionState) Provision(ctx context.Context) (func() error, error) {
	var err error

	// do the provisioning
	p.killFunc, err = p.driver.Provision(ctx, p)
	if err != nil {
		return nil, err
	}

	// write the statefile to dir
	err = p.WriteStateFile()
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

// write statefile to state directory
func (p *ProvisionState) WriteStateFile() error {

	stateFile := path.Join(p.StateDir, "provision_state.json")
	log.Info("provisioner: writing statefile", "path", stateFile)

	file, err := os.Create(stateFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(p)
	if err != nil {
		return err
	}

	return nil
}
