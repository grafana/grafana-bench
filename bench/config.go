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
	Arch             string
	GrafanaCommit    string
	GrafanaINIPath   string
	TestSuiteVersion string

	// Artifacts
	BuildArtifactName string
	BuildArtifactPath string

	// Tells us whether we need to resolve build
	Resolved bool
}

func NewBencher() *Config {
	return &Config{
		ProjectRoot:      utils.GetWorkdir(),
		Arch:             os.Getenv("ARCH"),
		GrafanaCommit:    os.Getenv("COMMIT"),
		GrafanaINIPath:   os.Getenv("INI"),
		TestSuiteVersion: os.Getenv("TEST_SUITE_VERSION"),
		Resolved:         false,
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

	if err := b.ResolveINI(); err != nil {
		return err
	}

	// Set artifacts to be used later
	b.BuildArtifactName = fmt.Sprintf("grafana-server-%s-%s", b.GrafanaCommit, strings.Replace(b.Arch, "/", "-", -1))
	b.BuildArtifactPath = path.Join("artifacts", b.BuildArtifactName)

	b.Resolved = true

	return nil
}
