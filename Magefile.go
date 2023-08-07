//go:build mage
// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/grafana/grafana-bench/bench"
	"github.com/grafana/grafana-bench/bench/buildcache"
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

// Build builds a grafana binary and stores it in the artifacts folder
// usage: GRAFANA_REVISION=branch:k8s-proof-of-concept mage buildcommit
func Build(ctx context.Context) error {
	build, err := BenchService.Builder.New(ctx, grafanaRevision, grafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run a build of grafana. If the build is not in the cache, it will
// authmatically do the build.
// If you use this command with the environment variable PROVISION=local this
// will block until you exit which will shut down the local grafana process.
func Run(ctx context.Context) error {
	if provisionDriver != provisioner.Local {
		fmt.Println("Provision driver is not local, defaulting to linux/amd64")
		grafanaArch = "linux/amd64"
	}

	// create a build with some defaults do the build if it's not resolved
	build, err := BenchService.Builder.New(ctx, grafanaRevision, grafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	// verify build exists in the cache
	resolved, err := BenchService.BuildCache.Resolve(ctx, buildcache.TypeBuild, build.ArtifactBuildName)
	if err != nil {
		return err
	}

	if resolved {
		fmt.Println("mage: build in cache")
	}

	ps, err := BenchService.Provisioner.New(ctx, provisionDriver, build, true)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	ps.WaitForReady(ctx)

	// Wait for signal to kill grafana if we're using the local driver
	if provisionDriver == provisioner.Local {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		fmt.Println("Shutting down grafana process")
	}

	return nil
}

// Bench handles building, running, and benchmarking a commit.
// Defaults to using latest commit on Main.
// You can set the revision yourself. Usage:
// `GRAFANA_REVISION=branch:k8s-proof-of-concept mage bench`
// `GRAFANA_REVISION=commit:c116545e0ba005e10e318da96688bdae01439bf5 mage bench`
//
// If you would like to specify a custom configuration, you can either set the
// GRAFANA_CONFIG variable or place a custom.ini in the bench directory on disk
// usage: `INI=custom.ini mage bench`
func Bench(ctx context.Context, testSuite string) error {
	if provisionDriver != provisioner.Local {
		fmt.Println("Provision driver is not local, defaulting to linux/amd64")
		grafanaArch = "linux/amd64"
	}

	// create a build with some defaults do the build if it's not resolved
	build, err := BenchService.Builder.New(ctx, grafanaRevision, grafanaArch)
	if err != nil {
		return err
	}
	if !build.Resolved {
		err := build.Run(ctx)
		if err != nil {
			return err
		}
	}

	// verify build exists in the cache
	resolved, err := BenchService.BuildCache.Resolve(ctx, buildcache.TypeBuild, build.ArtifactBuildName)
	if err != nil {
		return err
	}

	if resolved {
		fmt.Println("mage: build in cache")
	}

	ps, err := BenchService.Provisioner.New(ctx, provisionDriver, build, false)
	if err != nil {
		return err
	}

	killFunc, err := ps.Provision(ctx)
	if err != nil {
		return err
	}
	defer killFunc()

	ps.WaitForReady(ctx)

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "", testSuite, reportCloud)
	if err != nil {
		return err
	}

	// run the tests
	err = ps.RunTests(ctx, testRun)
	if err != nil {
		fmt.Println("error running tests:", err)
		fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}

	return err

	// remove the build artifacts
	//return ps.Destroy(ctx)
}

// Runs test suite on already running instance of grafana. Requires state for
// operation
func Test(ctx context.Context, testSuite string) error {
	if provisionState == "" {
		return fmt.Errorf("invalid state: \"%s\"", provisionState)
	}

	ps, err := BenchService.Provisioner.ReadStateFile(provisionState)
	if err != nil {
		return err
	}

	// test the build
	testRun, err := BenchService.Tester.New(ctx, "", testSuite, reportCloud)
	if err != nil {
		return err
	}

	// run the tests
	if err := ps.RunTests(ctx, testRun); err != nil {
		fmt.Println("error running tests:", err)
		fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}
	return nil
}

// Destroy looks up the state and tears down a provision state
func Destroy(ctx context.Context) error {
	if provisionState == "" {
		return fmt.Errorf("invalid state: \"%s\"", provisionState)
	}
	ps, err := BenchService.Provisioner.ReadStateFile(provisionState)
	if err != nil {
		return err
	}

	return ps.Destroy(ctx)
}

// Lists builds in cache
func ListBuilds(ctx context.Context) error {
	return BenchService.Builder.ListBuilds(ctx)
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

func HGTest(ctx context.Context, address, port, username, password, tests string) error {
	// create a new state
	provisionDriver = provisioner.HG
	ps, err := BenchService.Provisioner.New(ctx, provisionDriver, nil, false)
	if err != nil {
		return err
	}

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
		fmt.Println("error running tests:", err)
		fmt.Println("connectionString:", ps.K6Instance.GetConnectionString())
	}

	return nil
}
