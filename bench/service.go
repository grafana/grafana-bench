package bench

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

type BenchService struct {
	BuildCache  *buildcache.BuildCache
	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchServiceOrPanic(ctx context.Context) (*BenchService, *BenchServiceCfg) {
	cfg := GetBenchServiceCfgFromEnv(utils.Getwd())
	svc, err := NewBenchServiceFromConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}
	return svc, cfg
}

func NewBenchServiceFromConfig(ctx context.Context, bsc *BenchServiceCfg) (*BenchService, error) {
	buildCache, err := buildcache.NewBuildCache(ctx, bsc.buildCachePath, bsc.GCPCredPath, bsc.buildCacheBucket)
	if err != nil {
		return nil, err
	}

	vmEnabled := bsc.GCPCredPath != ""
	p, err := provisioner.NewProvisioner(ctx, buildCache, bsc.provisionerPath, vmEnabled, bsc.GCPCredPath, bsc.grafanaTmplPath)
	if err != nil {
		return nil, fmt.Errorf("error creating new provisioner: %w", err)
	}

	return &BenchService{
		BuildCache:  buildCache,
		Provisioner: p,
		Builder:     builder.NewBuildService(buildCache, bsc.builderPath),
		Tester:      tester.NewTester(ctx, bsc.testerPath, bsc.resultsPath, bsc.grafanaTestRepo, bsc.K6CloudProjectID, bsc.K6CloudToken),
	}, nil
}
