package bench

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// Build handles building a version of Grafana
func (b *Config) Build(ctx context.Context) error {
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

	// TODO if remotebuildcache exists, check to see if artifact is there and
	// upload if not. Do this in a non-blocking way so we can continue

	// copy build to local disk cache
	fmt.Println("copying executable to:", b.BuildArtifactPath)
	buildPath := path.Join(b.ProjectRoot, "build", "bin", b.Arch, "grafana")
	if err := sh.RunV("cp", buildPath, b.BuildArtifactPath); err != nil {
		return err
	}

	return nil
}
