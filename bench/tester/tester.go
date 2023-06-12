package tester

import "context"

type TesterService struct {
	LocalDir string
}

func NewTester(ctx context.Context, localDir string) *TesterService {
	return &TesterService{
		LocalDir: localDir,
	}
}
