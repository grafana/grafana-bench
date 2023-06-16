package provisioner

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
)

type ProvisionerService struct {
	BuildCache  *buildcache.BuildCache
	LocalDir    string
	VMEnabled   bool
	TemplateDir string
}

func NewProvisioner(ctx context.Context, localDir string, bc *buildcache.BuildCache, vmEnabled bool, templateDir string) (*ProvisionerService, error) {

	if localDir == "" {
		return nil, fmt.Errorf("provisioner: local directory cannot be empty")
	}

	if bc == nil {
		return nil, fmt.Errorf("provisioner: build cache cannot be nil")
	}

	if templateDir == "" {
		return nil, fmt.Errorf("provisioner: template directory cannot be empty")
	}

	return &ProvisionerService{
		LocalDir:    localDir,
		VMEnabled:   vmEnabled,
		TemplateDir: templateDir,
		BuildCache:  bc,
	}, nil
}

type ProvisionType string

const (
	Local  ProvisionType = "local"
	Remote ProvisionType = "remote"
)

func (p *ProvisionerService) New(ctx context.Context, t ProvisionType, build *builder.Build) (*ProvisionState, error) {

	if t == Remote && !p.VMEnabled {
		return nil, fmt.Errorf("Provisioner does not have VM support enabled")
	}

	uuid := uuid.Must(uuid.NewRandom())
	fmt.Println("provisioner: new state identifier:", uuid.String())

	workDir := path.Join(p.LocalDir, uuid.String(), "work")

	var driver *LocalDriver
	switch t {
	case Local:
		driver = NewLocalDriver(workDir, p.BuildCache)
	}

	state := &ProvisionState{
		driver:      driver,
		Identifier:  uuid.String(),
		Type:        t,
		WorkDir:     workDir,
		TemplateDir: p.TemplateDir,
		Build:       build,
	}

	err := os.MkdirAll(state.WorkDir, 0755)
	if err != nil {
		return nil, err
	}

	if t == Remote {
		state.StateDir = path.Join(p.LocalDir, uuid.String(), "state")
		err := os.MkdirAll(state.StateDir, 0755)
		if err != nil {
			return nil, err
		}
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
