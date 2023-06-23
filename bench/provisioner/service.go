package provisioner

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
)

type ProvisionerService struct {
	BuildCache             *buildcache.BuildCache
	TerraformTemplates     map[string]*template.Template
	LocalDir               string
	VMEnabled              bool
	GrafanaWorkDirTemplate string
	GCPCredentialsPath     string
}

func NewProvisioner(ctx context.Context, localDir string, bc *buildcache.BuildCache, vmEnabled bool, gcpCredentialsPath, grafanaWorkDirTemplate string) (*ProvisionerService, error) {

	if bc == nil {
		return nil, fmt.Errorf("provisioner: build cache cannot be nil")
	}

	if localDir == "" {
		return nil, fmt.Errorf("provisioner: local directory cannot be empty")
	}

	if grafanaWorkDirTemplate == "" {
		return nil, fmt.Errorf("provisioner: template directory cannot be empty")
	}

	templates, err := loadTerraformTemplates()
	if err != nil {
		return nil, fmt.Errorf("provisioner: error loading template: %w", err)
	}

	return &ProvisionerService{
		LocalDir:               localDir,
		VMEnabled:              vmEnabled,
		TerraformTemplates:     templates,
		GrafanaWorkDirTemplate: grafanaWorkDirTemplate,
		GCPCredentialsPath:     gcpCredentialsPath,
		BuildCache:             bc,
	}, nil
}

type ProvisionType string

const (
	Local ProvisionType = "local"
	GCP   ProvisionType = "gcp"
)

func (p *ProvisionerService) New(ctx context.Context, t ProvisionType, build *builder.Build) (*ProvisionState, error) {

	if t != Local && !p.VMEnabled {
		return nil, fmt.Errorf("Provisioner does not have VM support enabled")
	}

	uuid := uuid.Must(uuid.NewRandom())
	fmt.Println("provisioner: new state identifier:", uuid.String())

	localDir := path.Join(p.LocalDir, uuid.String())
	workDir := path.Join(localDir, "work")
	stateDir := path.Join(localDir, "state")

	fmt.Println("provisioner: local path:", localDir)

	var driver ProvisionDriver
	switch t {
	case Local:
		driver = NewLocalDriver(p.BuildCache)
	case GCP:
		driver = NewGCPDriver(p.BuildCache, p.TerraformTemplates, p.GCPCredentialsPath)
	}

	state := &ProvisionState{
		driver:      driver,
		Identifier:  uuid.String(),
		Type:        t,
		LocalDir:    localDir,
		StateDir:    stateDir,
		WorkDir:     workDir,
		TemplateDir: p.GrafanaWorkDirTemplate,
		Build:       build,
	}

	err := os.MkdirAll(state.WorkDir, 0755)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(state.StateDir, 0755)
	if err != nil {
		return nil, err
	}

	return state, nil
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
