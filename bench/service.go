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

	buildCache, err := buildcache.NewBuildCache(ctx, cfg.buildCachePath, cfg.GCPCredPath, cfg.buildCacheBucket)
	if err != nil {
		panic(fmt.Errorf("error instantiating build cache: %w", err))
	}

	provisioner, err := provisioner.NewProvisioner(ctx, buildCache, cfg.provisionerPath, cfg.GCPCredPath, cfg.grafanaTmplPath)
	if err != nil {
		panic(fmt.Errorf("error creating new provisioner: %w", err))
	}

	builder := builder.NewBuildService(buildCache, cfg.builderPath)

	tester := tester.NewTester(ctx,
		cfg.testerPath,
		cfg.testerUseCompiledTests,
		cfg.testerGrafanaTestRepo,
		cfg.K6CloudProjectID,
		cfg.K6CloudToken,
	)

	svc := &BenchService{
		BuildCache:  buildCache,
		Provisioner: provisioner,
		Builder:     builder,
		Tester:      tester,
	}

	return svc, cfg
}
