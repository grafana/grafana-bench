package bench

import (
	"log"
	"os"
	"path"

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
	GrafanaArch     string
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
	builderPath      string
	provisionerPath  string
	grafanaTmplPath  string
	testerPath       string
	grafanaTestRepo  string
	resultsPath      string
}

func GetBenchServiceCfgFromEnv(root string) *BenchServiceCfg {
	// load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env provided")
	}

	workPath := path.Join(root, "work")
	return &BenchServiceCfg{
		WorkPath:         workPath,
		GrafanaRevision:  env.EnvOrDefault("GRAFANA_REVISION", "branch:main"),
		GrafanaArch:      env.EnvOrDefault("GRAFANA_ARCH", env.GetLocalArch()),
		ProvisionDriver:  provisioner.ProvisionDriverFromString(env.EnvOrDefault("PROVISION", "local")),
		ProvisionState:   os.Getenv("STATE"),
		DestroyInfra:     env.EnvOrDefaultBool("DESTROY", "true"),
		SmokeTest:        env.EnvOrDefaultBool("SMOKE", "false"),
		ReportCloud:      env.EnvOrDefaultBool("REPORT_CLOUD", "false"),
		K6CloudProjectID: env.EnvOrDefault("K6_CLOUD_PROJECT", ""),
		K6CloudToken:     env.EnvOrDefault("K6_CLOUD_TOKEN", ""),
		GCPCredPath:      path.Join(root, "creds", env.EnvOrDefault("GCP_CREDS_FILE", "gcp.json")),

		// Build Cache
		buildCacheBucket: "bench-builds",
		buildCachePath:   path.Join(workPath, "buildcache"),

		// Builder
		builderPath: path.Join(workPath, "build"),

		// Provisioner
		provisionerPath: path.Join(workPath, "provision"),
		grafanaTmplPath: path.Join(workPath, "grafanaTemplate"),

		// Tester
		testerPath:      path.Join(workPath, "test"),
		grafanaTestRepo: "https://github.com/grafana/grafana-api-tests",
		resultsPath:     path.Join(workPath, "results"),
	}
}
