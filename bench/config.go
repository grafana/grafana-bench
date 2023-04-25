package bench

import (
	"os"

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

		Resolved: false,
	}

}

func (b *Config) ResolveConfig() error {
	if b.Resolved {
		return nil
	}

	if err := b.ResolveTestSuite(); err != nil {
		return err
	}

	if err := b.ResolveGrafanaCommit(); err != nil {
		return err
	}

	if err := b.ResolveArch(); err != nil {
		return err
	}

	if err := b.ResolveINI(); err != nil {
		return err
	}

	b.Resolved = true

	return nil
}
