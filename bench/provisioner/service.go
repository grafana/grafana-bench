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

func NewProvisioner(ctx context.Context, localDir string, bc *buildcache.BuildCache, vmEnabled bool, templateDir string) *ProvisionerService {
	return &ProvisionerService{
		LocalDir:    localDir,
		VMEnabled:   vmEnabled,
		TemplateDir: templateDir,
	}
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
		driver:     driver,
		Identifier: uuid.String(),
		Type:       t,
		WorkDir:    workDir,
		Build:      build,
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
		_, err := net.Dial("tcp", address)
		if err == nil {
			fmt.Println("Server is ready!")
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(time.Second)
	}
}
