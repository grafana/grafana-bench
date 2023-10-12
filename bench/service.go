package bench

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/builder"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

type BenchService struct {
	Log         *slog.Logger
	BuildCache  *buildcache.BuildCache
	Builder     *builder.BuilderService
	Provisioner *provisioner.ProvisionerService
	Tester      *tester.TesterService
}

func NewBenchServiceOrPanic(ctx context.Context, log *slog.Logger) (*BenchService, *BenchServiceCfg) {
	cfg := GetBenchServiceCfgFromEnv(utils.Getwd())

	buildCache, err := buildcache.NewBuildCache(ctx, log, cfg.buildCachePath, cfg.GCPCredPath, cfg.buildCacheBucket)
	if err != nil {
		panic(fmt.Errorf("error instantiating build cache: %w", err))
	}

	provisioner, err := provisioner.NewProvisioner(ctx, log, buildCache, cfg.provisionerPath, cfg.GCPCredPath, cfg.grafanaTmplPath)
	if err != nil {
		panic(fmt.Errorf("error creating new provisioner: %w", err))
	}

	builder := builder.NewBuildService(log, buildCache, cfg.builderPath)

	tester := tester.NewTester(ctx,
		log,
		cfg.testerPath,
		cfg.testerUseCompiledTests,
		cfg.testerGrafanaTestRepo,
		cfg.K6CloudProjectID,
		cfg.K6CloudToken,
	)

	svc := &BenchService{
		Log:         log,
		BuildCache:  buildCache,
		Provisioner: provisioner,
		Builder:     builder,
		Tester:      tester,
	}

	return svc, cfg
}
