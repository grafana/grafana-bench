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

	if err := b.ResolveBuildSuite(); err != nil {
		return err
	}

	exists, _ := utils.PathExists(b.BuildArtifactPath)
	if exists {
		fmt.Println("build artifacts cached, skipping build")
		return nil
	}

	// do the build
	err := utils.DoInDir(b.ProjectRoot, "build", func() error {
		ref := fmt.Sprintf("--grafana-ref=%s", b.GrafanaRevision)
		distro := fmt.Sprintf("--distro=%s", b.Arch)
		err := sh.RunV("go", "run", "./cmd", "--verbose", ref, "backend", "build", distro)
		return err
	})
	if err != nil {
		return err
	}

	// TODO add option to write remotely if doing a new build

	// copy build to artifact path
	// artifacts grafana, grafana-server, grafana-cli
	fmt.Println("copying executable to:", b.BuildArtifactPath)
	buildPath := path.Join(b.ProjectRoot, "build", "bin", b.Arch, "grafana")
	if err := sh.RunV("cp", buildPath, b.BuildArtifactPath); err != nil {
		return err
	}

	return nil
}
