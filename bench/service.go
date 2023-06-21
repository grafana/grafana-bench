package bench

import (
	"context"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
)

// Stores and handles setup of all services. Used when running bench as a single
// node
type BenchService struct {
	// Configured at runtime
	// TODO deprecate once we've removed mage functions
	ProjectRoot string
	BuildCache  *buildcache.BuildCache

	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchService(ctx context.Context, workPath, artifactsPath, GCSCredPath, bucketName string) (*BenchService, error) {

	// configure the cache
	buildCache, err := buildcache.NewBuildCache(ctx, artifactsPath, GCSCredPath, bucketName)
	if err != nil {
		return nil, err
	}

	// configure builder
	buildDir := path.Join(workPath, "build")
	b := builder.NewBuildService(buildDir, buildCache)

	// configure provisioner
	provisionDir := path.Join(workPath, "provision")
	grafanaTemplateDir := path.Join(workPath, "grafanaTemplate")
	p, err := provisioner.NewProvisioner(ctx, provisionDir, buildCache, false, grafanaTemplateDir)
	if err != nil {
		return nil, err
	}

	// configure tester
	testDir := path.Join(workPath, "test")
	t := tester.NewTester(ctx, testDir)

	return &BenchService{
		// deprecate
		ProjectRoot: workPath,
		BuildCache:  buildCache,
		Builder:     b,
		Provisioner: p,
		Tester:      t,
	}, nil
}
