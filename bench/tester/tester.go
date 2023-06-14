package tester

import (
	"context"

	"github.com/grafana/grafana-bench/bench/provisioner"
)

type TesterService struct {
	LocalDir string
}

func NewTester(ctx context.Context, localDir string) *TesterService {
	return &TesterService{
		LocalDir: localDir,
	}
}

type TestRun struct {
	Suite string `json:"suite"`
}

func (t *TesterService) New(ctx context.Context, ps *provisioner.ProvisionState) (*TestRun, error) {
	return nil, nil
}
