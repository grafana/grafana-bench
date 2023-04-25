package config

type BenchConfig struct {
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
