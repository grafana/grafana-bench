package bench

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
)

type BenchServiceCfg struct {
	// workpath is the directory on disk where service write working files
	WorkPath string

	GrafanaRevision  string
	GrafanaArch      string
	ProvisionDriver  provisioner.ProvisionType
	ProvisionState   string
	ReportCloud      bool
	K6CloudProjectID string
	K6CloudToken     string
	GCPCredPath      string

	buildCachePath  string
	builderPath     string
	provisionerPath string
	grafanaTmplPath string
	testerPath      string
	resultsPath     string
}

// Stores and handles setup of all services. Used when running bench as a single
// node
type BenchService struct {
	// Configured at runtime
	BuildCache *buildcache.BuildCache

	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchService(ctx context.Context, bsc *BenchServiceCfg) (*BenchService, error) {

	//workPath, artifactsPath, GCPCredPath, grafanaTestRepo, k6CloudToken, k6CloudProjectID, bucketName string)

	// configure the cache
	buildCacheBucket := "bench-builds"
	buildCachePath := path.Join(bsc.WorkPath, "buildcache")
	buildCache, err := buildcache.NewBuildCache(ctx, buildCachePath, bsc.GCPCredPath, buildCacheBucket)
	if err != nil {
		return nil, err
	}

	// configure builder
	buildDir := path.Join(bsc.WorkPath, "build")
	b := builder.NewBuildService(buildCache, buildDir)

	// configure provisioner
	provisionDir := path.Join(bsc.WorkPath, "provision")
	grafanaTemplateDir := path.Join(bsc.WorkPath, "grafanaTemplate")
	vmEnabled := bsc.GCPCredPath != ""
	p, err := provisioner.NewProvisioner(ctx, provisionDir, buildCache, vmEnabled, bsc.GCPCredPath, grafanaTemplateDir)
	if err != nil {
		return nil, fmt.Errorf("error creating new provisioner: %w", err)
	}

	// configure tester
	grafanaTestRepo := "https://github.com/grafana/grafana-api-tests"
	resultsDir := path.Join(bsc.WorkPath, "results")
	testDir := path.Join(bsc.WorkPath, "test")
	t := tester.NewTester(ctx, testDir, resultsDir, grafanaTestRepo, bsc.K6CloudProjectID, bsc.K6CloudToken)

	return &BenchService{
		BuildCache:  buildCache,
		Builder:     b,
		Provisioner: p,
		Tester:      t,
	}, nil
}
