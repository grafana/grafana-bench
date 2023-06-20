package provisioner

var _ ProvisionDriver = (*GCPDriver)(nil)

type GCPDriver struct {
	LocalDir string
	buildCache *buildcache.BuildCache
}

func NewLocalDriver(localDir string, buildCache *buildcache.BuildCache) *LocalDriver {
	return &LocalDriver{
		LocalDir:   localDir,
		buildCache: buildCache,
	}
}

func (l *LocalDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {
	return NilFunc, nil
}

// Blocking call that waits for grafana to become ready
func (l *LocalDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaAddress)
}

// Check - checks if Grafana + test runner are ready
func (l *LocalDriver) Ready(ctx context.Context, ps *ProvisionState) bool {
	return IsLive(ps.GrafanaAddress)
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (l *LocalDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	return nil
}

func (l *LocalDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	return nil
}
