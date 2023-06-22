package bench

import (
	"context"
	"os"
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
	// TODO deprecate ProjectRoot once we've removed mage functions
	ProjectRoot string
	BuildCache  *buildcache.BuildCache

	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchService(ctx context.Context, workPath, artifactsPath, GCPCredPath, k6CloudCredPath, bucketName string) (*BenchService, error) {

	// configure the cache
	buildCache, err := buildcache.NewBuildCache(ctx, artifactsPath, GCPCredPath, bucketName)
	if err != nil {
		return nil, err
	}

	// configure builder
	buildDir := path.Join(workPath, "build")
	b := builder.NewBuildService(buildDir, buildCache)

	// configure provisioner
	provisionDir := path.Join(workPath, "provision")
	grafanaTemplateDir := path.Join(workPath, "grafanaTemplate")
	vmEnabled := GCPCredPath != ""
	p, err := provisioner.NewProvisioner(ctx, provisionDir, buildCache, vmEnabled, GCPCredPath, grafanaTemplateDir)
	if err != nil {
		return nil, err
	}

	// configure tester
	resultsDir := path.Join(workPath, "results")
	testDir := path.Join(workPath, "test")
	k6cloudtoken := ""
	if k6CloudCredPath != "" {
		tokenBytes, err := os.ReadFile(k6CloudCredPath)
		if err != nil {
			return nil, err
		}
		k6cloudtoken = string(tokenBytes)
	}

	t := tester.NewTester(ctx, testDir, resultsDir, k6cloudtoken)

	return &BenchService{
		// deprecate
		ProjectRoot: workPath,
		BuildCache:  buildCache,
		Builder:     b,
		Provisioner: p,
		Tester:      t,
	}, nil
}
