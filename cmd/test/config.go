// Package config defines the test configuration
package test

import (
	"time"

	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/env"
)

type BenchConfig struct {
	BenchRevision      string
	MachineSpec        string
	Type               string
	Trigger            string
	EnvVars            map[string]string
	LogLevel           string
	ReportFormat       string
	Verbose            bool
	SlackNotifications bool
	DashboardURL       string
	Grafana            GrafanaConfig
	K6                 K6Config
	PW                 PWConfig
	Suite              TestSuiteConfig
	Slack              SlackNotifierConfig
}

type GrafanaConfig struct {
	Url      string
	UserName string
	Password string
	Timeout  time.Duration
}

type K6Config struct {
	CloudToken     string
	CloudProjectId string
	CloudOutput    bool
}

type PWConfig struct {
	PrepareCmd string
	ExecuteCmd string
}

type TestSuiteConfig struct {
	Name         string
	Repo         string
	RepoToken    string
	RepoDirs     []string
	Path         string
	BaseDir      string
	Revision     string
	RevisionFile string
	TestRunner   string
}

type SlackNotifierConfig struct {
	CodeownersMap string
	Token         string
}

// MergeEnv updates the config by overriding some fields with environment variables.
// Any environment variables that are not set will not override the config fields.
func MergeEnv(c *BenchConfig) {
	if c.BenchRevision == "" {
		c.BenchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
	}

	// Suite
	c.Suite.Revision = env.EnvOrDefault("TEST_SUITE_REVISION", c.Suite.Revision)
	c.Suite.Name = env.EnvOrDefault("TEST_SUITE_NAME", c.Suite.Name)

	// Grafana
	c.Grafana.Url = env.EnvOrDefault("GRAFANA_URL", c.Grafana.Url)
	c.Grafana.UserName = env.EnvOrDefault("GRAFANA_USER", c.Grafana.UserName)
	c.Grafana.Password = env.EnvOrDefault("GRAFANA_PASSWORD", c.Grafana.Password)

	// k6 config
	c.K6.CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", c.K6.CloudToken)
	c.K6.CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", c.K6.CloudProjectId)

	c.Suite.RepoToken = env.EnvOrDefault("TEST_SUITE_REPO_TOKEN", c.Suite.RepoToken)

	c.Slack.Token = env.EnvOrDefault("SLACK_TOKEN", c.Slack.Token)
}
