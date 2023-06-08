package bench

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// START HERE: test me. get builds working again
// add lifecycle policy to buildcache prefix in bucket
// start working on state directory for a run

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
	err = b.BuildCache.CacheBuild(ctx, buildPath, b.BuildArtifactName)
	if err != nil {
		return err
	}

	return nil
}
