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
	ProjectRoot string
	BuildCache  *buildcache.BuildCache

	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchService(ctx context.Context, projectRoot, artifactsPath, GCSCredPath, bucketName string) (*BenchService, error) {

	// configure the cache
	buildCache, err := buildcache.NewBuildCache(ctx, artifactsPath, GCSCredPath, bucketName)
	if err != nil {
		return nil, err
	}

	// configure builder
	buildDir := path.Join(projectRoot, "build")
	b := builder.NewBuildService(buildDir, buildCache)

	// configure provisioner
	provisionDir := path.Join(projectRoot, "provision")
	templateDir := path.Join(projectRoot, "templates")
	p, err := provisioner.NewProvisioner(ctx, provisionDir, buildCache, false, templateDir)
	if err != nil {
		return nil, err
	}

	// configure tester
	testDir := path.Join(projectRoot, "test")
	t := tester.NewTester(ctx, testDir)

	return &BenchService{
		ProjectRoot: projectRoot,
		BuildCache:  buildCache,
		Builder:     b,
		Provisioner: p,
		Tester:      t,
	}, nil
}
