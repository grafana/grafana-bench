package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
)

// This file is a thin wrapper to get us a quick CLI using mage.
// If you're adding or changing logic, that should happen in the bench/ package

var (
	// execRoot is the root of the repo or binary execution
	execRoot = utils.Getwd()
	// workpath is where all nested services will do their work
	workPath = path.Join(execRoot, "work")
	// buildcache is the location for the local buildcache
	buildCachePath = path.Join(workPath, "buildcache")

	// Get GoEnv from system running mage
	goEnv           = utils.GetCompilerEnvInfo()
	grafanaRevision = envOrDefault("GRAFANA_REVISION", "branch:main")
	grafanaArch     = envOrDefault("GRAFANA_ARCH", getLocalArch())
	provisionDriver = provisioner.ProvisionDriverFromString(envOrDefault("PROVISION", "local"))
	provisionState  = os.Getenv("STATE")
	reportCloud     = envOrDefaultBool("REPORT_CLOUD", "false")
	grafanaTestRepo = envOrDefault("GRAFANA_TEST_REPO_URL", "https://github.com/grafana/grafana-api-tests")

	// default k6 cloud token to jefflevinslunch instance
	k6CloudTokenPath = envOrDefault("K6_CLOUD_TOKEN_PATH", readK6Token(reportCloud, path.Join(execRoot, "creds", "k6cloud_ops_grafana_ops_net")))

	// default k6 cloud project: https://jefflevinslunch.grafana.net/a/k6-app/projects/3641403
	k6CloudProjectID = envOrDefault("K6_CLOUD_PROJECT", "3641403")

	// default to infra manager cred file
	gcsCredPath = path.Join(execRoot, "creds", "GCP-infra-manager-828bbfa6f427.json")

	// Setup bench service with defaults for CLI
	BenchService *bench.BenchService = CLIServiceDefaults(context.Background())
)

// CLIServiceDefaults setups up defaults for running bench
func CLIServiceDefaults(ctx context.Context) *bench.BenchService {
	svc, err := bench.NewBenchService(ctx, workPath, buildCachePath, gcsCredPath, grafanaTestRepo, k6CloudTokenPath, k6CloudProjectID, "bench-builds")
	if err != nil {
		panic(err)
	}
	return svc
}

// Gets the architecture of the machine running Bench
func getLocalArch() string {
	sys_os := goEnv["GOOS"]
	sys_arch := goEnv["GOARCH"]
	return fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
}

// Get environment variable or use default value
func envOrDefault(environmentVarName, defaultValue string) string {
	v := os.Getenv(environmentVarName)
	if v == "" {
		return defaultValue
	}

	return v
}

// Get boolean environment variable. panics if there's an issue with conversion
func envOrDefaultBool(environmentVarName, defaultValue string) bool {
	bool, err := strconv.ParseBool(envOrDefault("REPORT_CLOUD", "false"))
	if err != nil {
		panic(fmt.Sprintf("error reading bool env variable %s: %s", environmentVarName, err))
	}
	return bool
}

// reads a k6 token from a file if REPORT_CLOUD is true. Panics if there is a problem.
func readK6Token(reportCloud bool, path string) string {
	if !reportCloud || path == "" {
		return ""
	}

	// read in file
	tokenBytes, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("Error reading k6 cloud token: %s", err))
	}
	return strings.TrimSpace(string(tokenBytes))
}

func main() {
	if len(os.Args) != 6 {
		panic("need 6 args; address port username password tests")
	}

	var (
		ctx      = context.Background()
		address  = os.Args[1]
		port     = os.Args[2]
		username = os.Args[3]
		password = os.Args[4]
		tests    = os.Args[5]
	)

	if err := hgtest(ctx, address, port, username, password, tests); err != nil {
		panic(err)
	}
}

func hgtest(ctx context.Context, address, port, username, password, tests string) error {
	log.Println("hgtest")
	// create a new state
	provisionDriver = provisioner.HG
	ps, err := BenchService.Provisioner.New(ctx, provisionDriver, nil, false)
	if err != nil {
		return err
	}
	log.Println("yo")
	// populate grafana vm
	ps.GrafanaInstance = &provisioner.VMInstance{
		Address:         address,
		ServicePort:     port,
		GrafanaUser:     username,
		GrafanaPassword: password,
	}

	ps.WaitForReady(ctx)

	// set project id to https://jefflevinslunch.grafana.net/a/k6-app/projects/3653020
	BenchService.Tester.K6CloudProjectId = "3653020"
	// set token to jefflevinslunch
	BenchService.Tester.K6CloudToken = readK6Token(true, path.Join("creds", "k6cloud_jefflevinslunch_grafana_net"))

	// Hosted Grafana driver won't resolve. It will just make sure tests exist
	// where they're supposed to
	testRun, err := BenchService.Tester.New(ctx, "jalevin/test", tests, true)
	if err != nil {
		return err
	}

	// run the tests
	if err := ps.RunTests(ctx, testRun); err != nil {
		log.Println("error running tests:", err)
		log.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}

	return nil
}
