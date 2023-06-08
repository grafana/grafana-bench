package bench

import (
	"context"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/terraformer"
)

// Stores and handles setup of all services
type BenchService struct {
	// Configured at runtime
	ProjectRoot string

	Terraformer *terraformer.Terraformer
	BuildCache  *buildcache.BuildCache
}

func NewBenchService(ctx context.Context, projectRoot, artifactsPath, GCSCredPath, bucketName string) (*BenchService, error) {

	buildCache, err := buildcache.NewBuildCache(ctx, artifactsPath, GCSCredPath, bucketName)
	if err != nil {
		return nil, err
	}

	return &BenchService{
		ProjectRoot: projectRoot,
		BuildCache:  buildCache,
	}, nil
}
