package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
)

type ProvisionType string

const (
	Local ProvisionType = "local"
	GCP   ProvisionType = "gcp"
	HG    ProvisionType = "hg"
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

func (p *ProvisionerService) New(ctx context.Context, t ProvisionType, build *builder.Build, writeState bool) (*ProvisionState, error) {
	fmt.Printf("provisioner: using driver %s\n", t)

	if t != Local && !p.VMEnabled {
		return nil, fmt.Errorf("Provisioner does not have VM support enabled")
	}

	uuid := uuid.Must(uuid.NewRandom())
	fmt.Println("provisioner: new state identifier:", uuid.String())

	localDir := path.Join(p.LocalDir, uuid.String())
	workDir := path.Join(localDir, "work")
	stateDir := path.Join(localDir, "state")

	fmt.Println("provisioner: local path:", localDir)

	driver := p.InitDriver(t)

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

	// exit if not writing state
	if !writeState {
		fmt.Println("provisioner: writeState set to false. skip writing to disk")
		return state, nil
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

// Initializes provision driver from ProvisionType
func (p *ProvisionerService) InitDriver(t ProvisionType) ProvisionDriver {
	switch t {
	case Local:
		return NewLocalDriver(p.BuildCache)
	case GCP:
		return NewGCPDriver(p.BuildCache, p.TerraformTemplates, p.GCPCredentialsPath)
	case HG:
		return NewHGDriver()
	default:
		panic(fmt.Errorf("provisioner: unknown provision type: %s", t))
	}
}

func ProvisionDriverFromString(driverString string) ProvisionType {
	driverString = strings.ToLower(driverString)
	switch driverString {
	case "local":
		return Local
	case "gcp":
		return GCP
	case "hg":
		return HG
	default:
		panic(fmt.Errorf("provisioner: unknown provision type: %s", driverString))
	}
}

// Reads statefile from directory
func (p *ProvisionerService) ReadStateFile(stateIdentifier string) (*ProvisionState, error) {
	stateFile := path.Join(p.LocalDir, stateIdentifier, "state", "provision_state.json")
	fmt.Println("provisioner: reading statefile:", stateFile)
	file, err := os.Open(stateFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var ps ProvisionState
	err = decoder.Decode(&ps)
	if err != nil {
		return nil, err
	}

	ps.driver = p.InitDriver(ps.Type)

	// TODO the rest of this is dependent on the driver.
	// offload this part to the driver that handled the provisioning
	if ps.Type != Local {
		ps.GrafanaInstance, err = readVM(ps.StateDir, "grafana")
		if err != nil {
			return nil, err
		}

		ps.K6Instance, err = readVM(ps.StateDir, "k6")
		if err != nil {
			return nil, err
		}
	}

	return &ps, nil
}
