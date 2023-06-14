package bench

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
)

// Stores config for a given run of GrafanaBench
type BenchRun struct {
	*BenchService `json:"-"`

	// Stores information about the go environment
	// TODO do something with this later... probably only relevant when doing a
	// build
	GoEnv map[string]string

	// From environment
	Arch string `json:"arch"`
	// Branch or commit of Grafana to run. prefix the type that you're going to
	// provide. e.g. "branch:k8s-proof-of-concept" or "commit:e74e7fa"
	// commit refs must be 7 characters or longer
	GrafanaRevision string `json:"grafanaRevision"`
	// Path to custom.ini config to boot grafana with
	GrafanaINIPath string `json:"grafanaINIPath"`

	// Branch or commit of the test suite to run
	TestSuiteRevision string `json:"testSuiteRevision"`
	// File or folder name in github.com/grafana/grafana-api-tests/tests/
	TestSuite string `json:"testSuite"`
	// Directory to output files. This should be an absolute path on disk
	TestSummaryDir string `json:"testSummaryDir"`

	// Artifacts
	BuildArtifactName string `json:"buildArtifactName"`
	BuildArtifactPath string `json:"buildArtifactPath"`

	// Tells us whether we need to resolve config
	Resolved bool `json:"resolved"`
}

func NewBenchRun(ctx context.Context, svc *BenchService) *BenchRun {
	if svc == nil {
		panic("BenchService cannot be nil")
	}

	return &BenchRun{
		BenchService:      svc,
		Arch:              os.Getenv("ARCH"),
		GrafanaRevision:   os.Getenv("GRAFANA_REVISION"),
		GrafanaINIPath:    os.Getenv("GRAFANA_CONFIG"),
		TestSuiteRevision: os.Getenv("TEST_REVISION"),
		TestSuite:         os.Getenv("TEST_SUITE"),
		TestSummaryDir:    os.Getenv("TEST_SUMMARY_DIR"),
		Resolved:          false,
	}
}

// ResolveConfig resolves config resolves all environment variables for the config
func (b *BenchRun) ResolveConfig(ctx context.Context) error {
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

	// Set default place to output test results
	if b.TestSummaryDir == "" {
		b.TestSummaryDir = path.Join(b.ProjectRoot, "summary")
	}

	b.Resolved = true

	return nil
}

// ResolveGrafanaBuild will check the local cache and remote cache to for a
// build. If none exists, it will trigger a build. You can override build
// behavior by setting build to false.
func (b *BenchRun) ResolveGrafanaBuild(ctx context.Context, build bool) error {
	fmt.Println("build-cache: resolving grafana build")
	exists, err := b.BuildCache.Resolve(ctx, buildcache.BuildObj, b.BuildArtifactName)
	if err != nil {
		fmt.Println("build-cache: error resolving build:", err)
	}

	if !exists && build {
		fmt.Println("build-cache: build", b.BuildArtifactName, "not found. Building...")
		return b.Build(ctx)
	}

	fmt.Println("build-cache: build found on disk")

	return nil
}
