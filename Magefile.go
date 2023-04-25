//go:build mage
// +build mage

package main

// TODO
// test using mg.Deps to set BenchConfig on context and retrieve it or set

// embed and test setup functions
// embed and test dep functions
// test running suite
//

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	//mage:import setup
	"github.com/grafana/grafana-bench/pkg/deps"
	"github.com/grafana/grafana-bench/pkg/utils"
	"github.com/magefile/mage/sh"
)

var (
	Bencher config.BenchConfig = config.BenchConfig{
		ProjectRoot: utils.GetWorkdir(),
		GoEnv:       utils.GetCompilerEnvInfo(),

		Arch:             os.Getenv("ARCH"),
		GrafanaCommit:    os.Getenv("COMMIT"),
		GrafanaINIPath:   os.Getenv("INI"),
		TestSuiteVersion: os.Getenv("TEST_SUITE_VERSION"),

		Resolved: false,
	}
)

// Resolve architecture and artifact names
func ResolveArch() error {
	if Bencher.Arch == "" {
		// TODO maybe handle the case where key is not found
		sys_os := Bencher.GoEnv["GOOS"]
		sys_arch := Bencher.GoEnv["GOARCH"]

		Bencher.Arch = fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
	}
	fmt.Println("using arch:", Bencher.Arch)

	Bencher.BuildArtifactName = fmt.Sprintf("grafana-server-%s-%s", Bencher.GrafanaCommit, strings.Replace(Bencher.Arch, "/", "-", -1))
	Bencher.BuildArtifactPath = path.Join("artifacts", Bencher.BuildArtifactName)
	return nil
}

// Resolves dependencies like GrafanaCommit, Architecture, and Custom.ini for
// running the test suite
// TODO, not all of these need to run every time. Split these apart when
// optimizing
func SetDependencies() error {
	// check context for benchConfig
	// if there, check for resolved
	// if resolved, return
	// handle resolving
	// add to context

	if Bencher.Resolved {
		return nil
	}

	if err := ResolveGrafanaCommit(Bencher.ProjectRoot); err != nil {
		return err
	}

	if err := ResolveArch(); err != nil {
		return err
	}

	if err := ResolveINI(); err != nil {
		return err
	}

	Bencher.Resolved = true
	return nil
}

// Build builds a grafana binary and stores it in the artifacts folder
// usage: COMMIT=k8s-proof-of-concept mage buildcommit
func Build() error {
	mg.Deps(SetDependencies)

	if err := SetDependencies(); err != nil {
		return err
	}

	if err := deps.ResolveTestSuite(Bencher.ProjectRoot, Bencher.TestSuiteVersion); err != nil {
		return err
	}

	exists, _ := utils.PathExists(Bencher.BuildArtifactPath)
	if exists {
		fmt.Println("build artifacts cached, skipping build")
		return nil
	}

	// do the build
	err := utils.DoInDir(Bencher.ProjectRoot, "build", func() error {
		ref := fmt.Sprintf("--grafana-ref=%s", Bencher.GrafanaCommit)
		distro := fmt.Sprintf("--distro=%s", Bencher.Arch)
		err := sh.RunV("go", "run", "./cmd", "--verbose", ref, "backend", "build", distro)
		return err
	})
	if err != nil {
		return err
	}

	// copy build to artifact path
	// artifacts grafana, grafana-server, grafana-cli
	fmt.Println("copying executable to:", Bencher.BuildArtifactPath)
	buildPath := path.Join(Bencher.ProjectRoot, "build", "bin", Bencher.Arch, "grafana")
	if err := sh.RunV("cp", buildPath, Bencher.BuildArtifactPath); err != nil {
		return err
	}

	return nil
}

// BenchCommit load tests a commit.
// If you don't set the commit environment variable, it will
// default to main and resolve the git hash. You can also set commit to
// be a branch and it will grab the latest commit for that branch.
// usage:
//
// COMMIT=k8s-proof-of-concept mage bench
//
// COMMIT=c116545e0ba005e10e318da96688bdae01439bf5 mage bench
//
// By default we will look for a custom.ini in the project root, however, you
// can also specify this by environment variable and path.
//
// usage: INI=custom.ini mage bench
func Bench() error {
	var err error

	if err := SetDependencies(); err != nil {
		return err
	}

	// do the build if we need it
	if err := Build(); err != nil {
		return err
	}

	fmt.Println("setting up work directory")

	// delete old workdir if exists
	if err := sh.RunV("rm", "-rf", path.Join(Bencher.ProjectRoot, "work")); err != nil {
		return err
	}

	// copy template directory
	templateConf := path.Join(Bencher.ProjectRoot, "templates")
	workConfPath := path.Join(Bencher.ProjectRoot, "work")
	if err := sh.RunV("cp", "-r", templateConf, workConfPath); err != nil {
		return err
	}

	// get default.ini for that commit
	iniArtifact := fmt.Sprintf("%s_defaults.ini", Bencher.GrafanaCommit)
	iniArtifactPath := path.Join(Bencher.ProjectRoot, "artifacts", iniArtifact)
	exists, _ := utils.PathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", Bencher.GrafanaCommit)
		if err := sh.RunV("curl", iniUrl, "-o", iniArtifactPath); err != nil {
			return err
		}
	}

	// copy ini to workdir
	iniWorkPath := path.Join(Bencher.ProjectRoot, "work", "conf", "defaults.ini")
	if err := sh.RunV("cp", iniArtifactPath, iniWorkPath); err != nil {
		return err
	}

	// copy custom.ini into work dir
	if Bencher.GrafanaINIPath != "" {
		fmt.Println("found custom.ini")
		customIniWorkPath := path.Join(Bencher.ProjectRoot, "work", "conf", "custom.ini")
		if err := sh.Run("cp", Bencher.GrafanaINIPath, customIniWorkPath); err != nil {
			return err
		}
	}

	// copy artifact
	workExecutable := path.Join(Bencher.ProjectRoot, "work", Bencher.BuildArtifactName)
	if err := sh.RunV("cp", Bencher.BuildArtifactPath, workExecutable); err != nil {
		return err
	}

	// boot grafana
	fmt.Println("booting grafana")
	cmd := exec.Command(workExecutable, "server")
	err = utils.DoInDir(Bencher.ProjectRoot, "work", func() error {
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting server:", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// make sure we kill grafana
	defer func() error {
		return cmd.Process.Kill()
	}()

	// Wait for the server to start up
	for {
		_, err := net.Dial("tcp", "localhost:3000")
		if err == nil {
			fmt.Println("Server is ready!")
			break
		}
		fmt.Println("Waiting for server...")
		time.Sleep(time.Second)
	}

	// get featureToggles & buildInfo from response /api/frontend/settings
	// only contains list of things that are turned on

	// run k6 tests
	err = utils.DoInDir(Bencher.ProjectRoot, "tests", func() error {
		if err := sh.RunV("k6", "run", "tests/dashboards.js"); err != nil {
			// k6 run tests/tests/dashboards.js
			return err
		}

		return nil
	})

	return err
}

// Resolve branch to latest commit of branch
func ResolveGrafanaCommit(commit string) error {
	if commit == "" {
		commit = "main"
	}

	// if already a commit hash, we can just return
	if utils.IsCommitHash(commit) {
		fmt.Println("using commit:", commit)
		return nil
	}

	// get latest commit from branch
	branch := commit
	fmt.Println("branch:", branch, "specified. Resolving latest commit")
	commit, err := utils.GetLatestBranchCommit("https://github.com/grafana/grafana", commit)
	if err != nil {
		return err
	}
	fmt.Printf("branch: %s resolved to `%s`\n", branch, commit)
	return nil
}

// ResolveINI determines if there is a custom.ini to test a version of grafana
// with
func ResolveINI() error {
	// check if INI is set
	if Bencher.GrafanaINIPath != "" {
		if !filepath.IsAbs(Bencher.GrafanaINIPath) {
			Bencher.GrafanaINIPath = path.Join(Bencher.ProjectRoot, Bencher.GrafanaINIPath)
		}

		exists, _ := utils.PathExists(Bencher.GrafanaINIPath)
		if exists {
			return nil
		}
	}

	// check if custom.ini in project root
	dirIni := path.Join(Bencher.ProjectRoot, "custom.ini")
	exists, _ := utils.PathExists(dirIni)
	if exists {
		Bencher.GrafanaINIPath = dirIni
		return nil
	}

	return nil
}
