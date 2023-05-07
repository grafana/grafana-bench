package bench

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
)

type Config struct {
	// Configured at runtime
	ProjectRoot string
	GoEnv       map[string]string

	// From environment
	Arch string
	// Branch or commit of Grafana to run. prefix the type that you're going to
	// provide. e.g. "branch:k8s-proof-of-concept" or "commit:e74e7fa"
	// commit refs must be 7 characters or longer
	GrafanaRevision string
	// Path to custom.ini config to boot grafana with
	GrafanaINIPath string

	// Branch or commit of the test suite to run
	TestSuiteRevision string
	// File or folder name in github.com/grafana/grafana-api-tests/tests/
	TestSuite string
	// Directory to output files. This should be an absolute path on disk
	TestSummaryDir string

	// Credentials to remote build cache
	RemoteBuildCacheCredentials string
	RemoteBuildCacheBucket      string
	RemoteBuildCache            *buildcache.BuildCache

	// Artifacts
	BuildArtifactName string
	BuildArtifactPath string

	// Tells us whether we need to resolve config
	Resolved bool
}

func NewBencher() *Config {
	return &Config{
		ProjectRoot:                 utils.GetWorkdir(),
		Arch:                        os.Getenv("ARCH"),
		GrafanaRevision:             os.Getenv("GRAFANA_REVISION"),
		GrafanaINIPath:              os.Getenv("GRAFANA_CONFIG"),
		TestSuiteRevision:           os.Getenv("TEST_REVISION"),
		TestSuite:                   os.Getenv("TEST_SUITE"),
		TestSummaryDir:              os.Getenv("TEST_SUMMARY_DIR"),
		RemoteBuildCacheCredentials: "GCP-infra-manager-828bbfa6f427.json",
		RemoteBuildCacheBucket:      "bench-builds",
		RemoteBuildCache:            nil,
		Resolved:                    false,
	}
}

// ResolveConfig resolves config resolves all environment variables for the config
func (b *Config) ResolveConfig(ctx context.Context) error {
	if b.Resolved {
		return nil
	}

	if err := b.CheckDeps(); err != nil {
		return err
	}

	if err := b.ResolveArch(); err != nil {
		return err
	}

	if err := b.ResolveGrafanaRevision(); err != nil {
		return err
	}

	if err := b.ResolveGrafanaINI(); err != nil {
		return err
	}

	if err := b.ResolveRemoteBuildCache(ctx); err != nil {
		return err
	}

	// Set default place to output test results
	if b.TestSummaryDir == "" {
		b.TestSummaryDir = path.Join(b.ProjectRoot, "summary")
	}

	b.Resolved = true

	return nil
}

// TODO START HERE
// 4. run mage bench and make sure it checks local cache
// 5. run mage bench and make sure it checks remote cache

// 6. start working on writing builds to buildcache if they don't already
// exist there
// 7. add lifecycle policy to buildcache prefix

// ResolveGrafanaBuild will check the local cache and remote cache to for a
// build. If none exists, it will trigger a build. You can override build
// behavior by setting build to false.
func (b *Config) ResolveGrafanaBuild(ctx context.Context, build bool) error {
	// check if target exists
	exists, _ := utils.PathExists(b.BuildArtifactPath)
	if exists {
		return nil
	}

	// no build cache configured, build it
	if b.RemoteBuildCache == nil {
		fmt.Println("build-cache: artifact not found on disk. No build cache. Building Grafana")
		if build {
			return b.Build(ctx)
		}
	}

	// download from cache or build
	fmt.Println("Artifact not found on disk. Checking remote cache")
	success, err := b.RemoteBuildCache.DownloadGrafanaBuild(ctx, b.BuildArtifactName, b.BuildArtifactPath)

	// downloaded. exit
	if success {
		return nil
	}

	// print the error
	if err != nil {
		fmt.Println("build-cache: Error contacting remote build cache:", err)
	}

	// Do the build
	fmt.Println("Building Grafana")
	if build {
		return b.Build(ctx)
	}
	return nil
}

func (b *Config) getBuildFromCache() {}
