package bench

import (
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils/env"
	"github.com/joho/godotenv"
)

type BenchServiceCfg struct {
	// workpath is the directory on disk where service write working files
	WorkPath string

	// Reference to OSS branch to be built. Defaults to `branch:main`. Can also
	// provide a git commit with lenght of 7 or longer `commit:1234567`
	GrafanaRevision string
	// Architecture for the build. windows, darwin, or linux formatted linux/amd64
	GrafanaArch string
	// Address of running Grafana instance. Defaults to http://localhost:3000
	GrafanaAddress string
	// Username of running Grafana instance. Defaults to admin
	GrafanaUser string
	// Password of running Grafana instance. Defaults to admin
	GrafanaPassword string
	// Driver to use when provisioning
	ProvisionDriver provisioner.ProvisionType
	// Determines whether to destroy the infra on exit
	DestroyInfra bool
	// Identifier for provision state. If provided, we try to use that state instead of
	// building + provisioning. Useful for running tests against a running
	// Grafana
	ProvisionState string
	// Flag changes k6 command to run a single iteration. true | false. default
	// false
	SmokeTest bool
	// Flag enables reporting to k6 cloud. true | false. default false
	ReportCloud bool
	// K6CloudProjectID string
	K6CloudProjectID string
	// K6CloudToken string
	K6CloudToken string
	// Path to GCP credentials
	GCPCredPath string

	buildCacheBucket string
	buildCachePath   string
	// working directory for builder
	builderPath string
	// working directory for provisioner
	provisionerPath string
	// path to grafana template when compiling backend only
	grafanaTmplPath string

	// working directory for tester
	testerPath string
	// determines whether tester should manage lifecycle of repo or use
	// precompiled tests
	testerUseCompiledTests bool
	// url for github repo with grafana tests
	testerGrafanaTestRepo string
}

func GetBenchServiceCfgFromEnv(log *slog.Logger, root string) *BenchServiceCfg {
	log.With("svc", "config")

	// load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Warn("No .env provided")
	}

	workPath := env.EnvOrDefault("WORK_PATH", path.Join(root, "work"))

	// If TEST_PATH does not being with a /, assume it's relative to
	// root of bench and append the work path
	testerPath := env.EnvOrDefault("TEST_PATH", path.Join(workPath, "test"))
	if !strings.HasPrefix(testerPath, "/") {
		testerPath = path.Join(root, testerPath)
	}

	return &BenchServiceCfg{
		// Base cfg
		WorkPath:         workPath,
		buildCacheBucket: "bench-builds",
		buildCachePath:   path.Join(workPath, "buildcache"),
		builderPath:      path.Join(workPath, "build"),
		provisionerPath:  path.Join(workPath, "provision"),
		grafanaTmplPath:  path.Join(workPath, "grafanaTemplate"),

		// Build settings
		GrafanaRevision: env.EnvOrDefault("GRAFANA_REVISION", "branch:main"),
		GrafanaArch:     env.EnvOrDefault("GRAFANA_ARCH", env.GetLocalArch()),

		// Provision settings
		ProvisionDriver: provisioner.ProvisionDriverFromString(env.EnvOrDefault("PROVISION", "local")),
		ProvisionState:  os.Getenv("STATE"),
		DestroyInfra:    env.EnvOrDefaultBool("DESTROY", "true"),
		GrafanaAddress:  env.EnvOrDefault("GRAFANA_ADDRESS", "http://localhost:3000"),
		GrafanaUser:     env.EnvOrDefault("GRAFANA_USER", "admin"),
		GrafanaPassword: env.EnvOrDefault("GRAFANA_PASSWORD", "admin"),

		// Test settings
		SmokeTest:              env.EnvOrDefaultBool("SMOKE", "false"),
		ReportCloud:            env.EnvOrDefaultBool("REPORT_CLOUD", "false"),
		K6CloudProjectID:       env.EnvOrDefault("K6_CLOUD_PROJECT_ID", ""),
		K6CloudToken:           env.EnvOrDefault("K6_CLOUD_TOKEN", ""),
		GCPCredPath:            path.Join(root, "creds", env.EnvOrDefault("GCP_CREDS_FILE", "gcp.json")),
		testerPath:             testerPath,
		testerUseCompiledTests: env.EnvOrDefaultBool("USE_COMPILED_TESTS", "false"),
		testerGrafanaTestRepo:  "https://github.com/grafana/grafana-api-tests",
	}
}
