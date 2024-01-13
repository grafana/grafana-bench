package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/buildcache"
)

type ProvisionType string

const (
	Local ProvisionType = "local"
	GCP   ProvisionType = "gcp"
	HG    ProvisionType = "hg"
)

type ProvisionerService struct {
	Log                    *slog.Logger
	BuildCache             *buildcache.BuildCache
	TerraformTemplates     map[string]*template.Template
	LocalDir               string
	GrafanaWorkDirTemplate string
	GCPCredentialsPath     string
}

func NewProvisionSvc(ctx context.Context, log *slog.Logger, bc *buildcache.BuildCache, localDir string, gcpCredentialsPath, grafanaWorkDirTemplate string) (*ProvisionerService, error) {
	log = log.With("svc", "provisioner")

	if bc == nil {
		return nil, fmt.Errorf("build cache cannot be nil")
	}

	if localDir == "" {
		return nil, fmt.Errorf("local directory cannot be empty")
	}

	if grafanaWorkDirTemplate == "" {
		return nil, fmt.Errorf("template directory cannot be empty")
	}

	templates, err := loadTerraformTemplates()
	if err != nil {
		return nil, fmt.Errorf("error loading template: %w", err)
	}

	return &ProvisionerService{
		Log:                    log,
		LocalDir:               localDir,
		TerraformTemplates:     templates,
		GrafanaWorkDirTemplate: grafanaWorkDirTemplate,
		GCPCredentialsPath:     gcpCredentialsPath,
		BuildCache:             bc,
	}, nil
}

func (p *ProvisionerService) New(ctx context.Context, t ProvisionType, grafanaRevision string, grafanaArch string, writeState bool) (*ProvisionState, error) {
	log := p.Log.With("driver", t)

	uuid := uuid.Must(uuid.NewRandom())
	p.Log.Info("new state identifier", "id", uuid.String())

	localDir := path.Join(p.LocalDir, uuid.String())
	workDir := path.Join(localDir, "work")
	stateDir := path.Join(localDir, "state")

	state := &ProvisionState{
		Log:         log,
		driver:      p.InitDriver(t),
		Identifier:  uuid.String(),
		Type:        t,
		LocalDir:    localDir,
		StateDir:    stateDir,
		WorkDir:     workDir,
		TemplateDir: p.GrafanaWorkDirTemplate,
		GrafanaBuildInfo: GrafanaBuildInfo{
			Revision: grafanaRevision,
			Arch:     grafanaArch,
		},
	}

	// exit if not writing state
	if !writeState {
		log.Info("writeState set to false. skip writing to disk")
		return state, nil
	}

	log.Info("local path", "dir", localDir)

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

// NewLocalDevState creates a provision state assuming Grafana is running on https://localhost:3000. Used for local development workflow.
// e.g. `mage test dashboards` without providing a state
func (p *ProvisionerService) NewLocalDevState(ctx context.Context, grafanaAddress string, grafanaUser string, grafanaPassword string) *ProvisionState {

	instance, _ := NewReadOnlyGrafanaVM(grafanaAddress, grafanaUser, grafanaPassword)
	return &ProvisionState{
		Log:              p.Log.With("driver", Local),
		driver:           p.InitDriver(Local),
		Identifier:       "LOCALDEVSTATE",
		Type:             Local,
		GrafanaBuildInfo: GrafanaBuildInfo{},
		GrafanaInstance:  instance,
	}
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
	p.Log.Info("reading statefile", "file", stateFile)
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

	ps.Log = p.Log.With("driver", ps.Type)
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
