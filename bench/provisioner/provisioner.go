package provisioner

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/builder"
)

type ProvisionerService struct {
	LocalDir    string
	VMEnabled   bool
	TemplateDir string
}

type ProvisionType string

const (
	Local  ProvisionType = "local"
	Remote ProvisionType = "remote"
)

func NewProvisioner(ctx context.Context, localDir string, vmEnabled bool, templateDir string) *ProvisionerService {
	return &ProvisionerService{
		LocalDir:    localDir,
		VMEnabled:   vmEnabled,
		TemplateDir: templateDir,
	}
}

func (p *ProvisionerService) New(ctx context.Context, t ProvisionType, build *builder.Build) (*ProvisionState, error) {

	if t == Remote && !p.VMEnabled {
		return nil, fmt.Errorf("Provisioner does not have VM support enabled")
	}

	uuid := uuid.Must(uuid.NewRandom())
	fmt.Println("provisioner: new state identifier:", uuid.String())

	state := &ProvisionState{
		Identifier: uuid.String(),
		Type:       t,
		WorkDir:    path.Join(p.LocalDir, uuid.String(), "work"),
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
