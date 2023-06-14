package bench

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Build handles building a version of Grafana
func (b *BenchRun) Build(ctx context.Context) error {
	if err := b.ResolveConfig(ctx); err != nil {
		return err
	}

	if err := b.ResolveBuildSuite(ctx); err != nil {
		return err
	}

	// do the build
	err := utils.DoInDir(b.ProjectRoot, "build", func() error {
		// Note, verbose and distro must be provided at the end of the command
		err := sh.RunV("go", "run", "./cmd", "backend", "build",
			fmt.Sprintf("--distro=%s", b.Arch),
			fmt.Sprintf("--grafana-ref=%s", b.GrafanaRevision),
			"--verbose")
		return err
	})
	if err != nil {
		return err
	}

	buildPath := path.Join(b.ProjectRoot, "build", "bin", b.Arch, "grafana")
	err = b.BuildCache.StoreFile(ctx, buildcache.BuildObj, buildPath, b.BuildArtifactName)
	if err != nil {
		return err
	}

	return nil
}

func (b *BenchRun) ListBuilds(ctx context.Context) error {
	builds, err := b.BuildCache.List(ctx, buildcache.BuildObj)
	if err != nil {
		return err
	}

	fmt.Println("Builds")
	for _, b := range builds {
		fmt.Printf("%s: %s\n", b.Location, b.Name)
	}

	return nil
}
