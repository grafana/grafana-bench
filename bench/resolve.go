package bench

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/grafana/grafana-bench/bench/utils/git"
)

func (b *Config) ResolveGrafanaRevision() error {
	if b.GrafanaRevision == "" {
		b.GrafanaRevision = "branch:main"
	}

	pieces := strings.Split(b.GrafanaRevision, ":")
	if len(pieces) != 2 {
		return fmt.Errorf("Invalid GrafanaRevision format. Use `commit:e74e7fa` or `branch:main`")
	}

	t, val := pieces[0], pieces[1]

	var commit string
	var err error
	if t == "branch" {
		fmt.Println("grafana: branch", val, "specified. Resolving latest commit")
		commit, err = git.ResolveLatestBranchCommit("grafana/grafana", val)
		if err != nil {
			return err
		}
	} else if t == "commit" {
		fmt.Println("grafana: commit", val, "specified. Resolving commit")
		commit, err = git.ResolveFullCommit("grafana/grafana", val)
		if err != nil {
			return err
		}
	}

	b.GrafanaRevision = commit
	fmt.Println("grafana: revision resolved to:", b.GrafanaRevision)
	return nil
}

func (b *Config) ResolveArch() error {
	// Get GoEnv from system
	b.GoEnv = utils.GetCompilerEnvInfo()

	// TODO maybe handle the case where key is not found. probably just a panic
	if b.Arch == "" {
		sys_os := b.GoEnv["GOOS"]
		sys_arch := b.GoEnv["GOARCH"]
		b.Arch = fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
	}

	fmt.Println("arch:", b.Arch)

	return nil
}

func (b *Config) ResolveGrafanaINI() error {
	// Check if INI environment set and use that
	// If no INI check to see if local custom.ini in root
	// else blank

	// check if INI is set
	if b.GrafanaINIPath != "" {
		if !filepath.IsAbs(b.GrafanaINIPath) {
			b.GrafanaINIPath = path.Join(b.ProjectRoot, b.GrafanaINIPath)
		}

		exists, _ := utils.PathExists(b.GrafanaINIPath)
		if !exists {
			return fmt.Errorf("grafana-config: error specified config file not found %s", b.GrafanaINIPath)
		}

		fmt.Println("grafana-config: resolved to", b.GrafanaINIPath)
		return nil
	}

	// check if custom.ini in project root
	dirIni := path.Join(b.ProjectRoot, "custom.ini")
	exists, _ := utils.PathExists(dirIni)
	if exists {
		b.GrafanaINIPath = dirIni
		fmt.Println("grafana-config: custom.ini found. using", b.GrafanaINIPath)
		return nil
	}

	fmt.Println("grafana-config: no config specified. using defaults")
	return nil
}

func (b *Config) ResolveRemoteBuildCache(ctx context.Context) error {
	if b.RemoteBuildCacheCredentials != "" && b.RemoteBuildCacheBucket != "" {
		rbc, err := buildcache.NewBuildCache(ctx, b.RemoteBuildCacheCredentials, b.RemoteBuildCacheBucket)
		if err != nil {
			return err
		}
		fmt.Println("build-cache: using GCS bucket:", b.RemoteBuildCacheBucket)
		b.RemoteBuildCache = rbc
	} else {
		fmt.Println("build-cache: not configured")
	}
	return nil
}
