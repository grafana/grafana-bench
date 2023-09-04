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

	GrafanaRevision  string
	GrafanaArch      string
	ProvisionDriver  provisioner.ProvisionType
	ProvisionState   string
	ReportCloud      bool
	K6CloudProjectID string
	K6CloudToken     string
	GCPCredPath      string

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
		log.Fatal("Error loading .env file")
	}

	workPath := path.Join(root, "work")
	return &BenchServiceCfg{
		WorkPath:         workPath,
		GrafanaRevision:  env.EnvOrDefault("GRAFANA_REVISION", "branch:main"),
		GrafanaArch:      env.EnvOrDefault("GRAFANA_ARCH", env.GetLocalArch()),
		ProvisionDriver:  provisioner.ProvisionDriverFromString(env.EnvOrDefault("PROVISION", "local")),
		ProvisionState:   os.Getenv("STATE"),
		ReportCloud:      env.EnvOrDefaultBool("REPORT_CLOUD", "false"),
		K6CloudProjectID: env.EnvOrDefault("K6_CLOUD_PROJECT", ""),
		K6CloudToken:     env.EnvOrDefault("K6_CLOUD_TOKEN_PATH", ""),
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
