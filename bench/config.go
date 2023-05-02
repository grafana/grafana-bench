package bench

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
)

type Config struct {
	// Configured at runtime
	ProjectRoot string
	GoEnv       map[string]string

	// From environment
	Arch string
	// Branch or commit of Grafana to run
	GrafanaRevision string
	// Path to custom.ini config to boot grafana with
	GrafanaINIPath string

	// Branch or commit of the test suite to run
	TestSuiteRevision string
	// File or folder name in github.com/grafana/grafana-api-tests/tests/
	TestSuite string

	// Artifacts
	BuildArtifactName string
	BuildArtifactPath string

	// Tells us whether we need to resolve config
	Resolved bool
}

func NewBencher() *Config {
	return &Config{
		ProjectRoot:       utils.GetWorkdir(),
		Arch:              os.Getenv("ARCH"),
		GrafanaRevision:   os.Getenv("GRAFANA_REVISION"),
		GrafanaINIPath:    os.Getenv("GRAFANA_CONFIG"),
		TestSuiteRevision: os.Getenv("TEST_REVISION"),
		TestSuite:         os.Getenv("TEST_SUITE"),
		Resolved:          false,
	}

}

// ResolveConfig resolves config resolves all environment variables for the config
func (b *Config) ResolveConfig() error {
	if b.Resolved {
		return nil
	}

	if err := b.CheckDeps(); err != nil {
		return err
	}

	if err := b.ResolveArch(); err != nil {
		return err
	}

	if err := b.ResolveGrafanaCommit(); err != nil {
		return err
	}

	if err := b.ResolveGrafanaINI(); err != nil {
		return err
	}

	// Set artifacts to be used later
	b.BuildArtifactName = fmt.Sprintf("grafana-server-%s-%s", b.GrafanaRevision, strings.Replace(b.Arch, "/", "-", -1))
	b.BuildArtifactPath = path.Join("artifacts", b.BuildArtifactName)

	b.Resolved = true

	return nil
}
