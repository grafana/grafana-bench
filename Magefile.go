//go:build mage
// +build mage

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

var (
	commit  string = os.Getenv("COMMIT")
	iniPath string = os.Getenv("INI")

	projectRoot string = getWorkdir()

	arch          string
	artifact_name string
	artifact_path string

	resolved bool = false
)

// Resolve branch to latest commit of branch
func ResolveCommit() error {
	if commit == "" {
		commit = "main"
	}

	// resolve branch
	if len(commit) != 40 {
		fmt.Println("no grafana commit specified using branch:", commit)
		b := commit
		if commit == "main" {
			b = "HEAD"
		}
		resolved, err := sh.Output("git", "ls-remote", "https://github.com/grafana/grafana", b, "-c7")
		if err != nil {
			return fmt.Errorf("Error resolving git commit %s: %s", commit, err)
		}

		resolved = strings.Split(resolved, "\t")[0]

		fmt.Printf("main resolved to `%s`\n", resolved)
		commit = resolved
		return nil
	}

	fmt.Println("using commit:", commit)
	return nil
}

// Resolve architecture and artifact names
func ResolveArch() error {
	arch = os.Getenv("ARCH")
	if arch == "" {
		sys_os, err := sh.Output("uname", "-s")
		if err != nil {
			return fmt.Errorf("error resolving OS %s", err)
		}

		sys_arch, err := sh.Output("uname", "-m")
		if err != nil {
			return fmt.Errorf("error resolve architecture %s", err)
		}
		arch = fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
	}
	fmt.Println("using arch:", arch)

	artifact_name = fmt.Sprintf("grafana-server-%s-%s", commit, strings.Replace(arch, "/", "-", -1))
	artifact_path = path.Join("artifacts", artifact_name)
	return nil
}

func ResolveINI() error {
	// check if INI is set
	if iniPath != "" {
		if !filepath.IsAbs(iniPath) {
			iniPath = path.Join(projectRoot, iniPath)
		}

		exists, _ := pathExists(iniPath)
		if exists {
			return nil
		}
	}

	// check if custom.ini in project root
	dirIni := path.Join(projectRoot, "custom.ini")
	exists, _ := pathExists(dirIni)
	if exists {
		iniPath = dirIni
		return nil
	}

	return nil
}

// bootstrap clones test and build repos locally
func Bootstrap() error {

	// check if test repo is cloned locally
	exists, err := pathExists("tests")
	if err != nil {
		return fmt.Errorf("Issue checking directory path")
	}

	if !exists {
		if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-api-tests", "tests"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}
	}

	// check if build repo cloned locally
	exists, err = pathExists("build")
	if err != nil {
		return fmt.Errorf("Issue checking directory path")
	}

	if !exists {
		if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-build", "build"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}
	}

	// ensure k6 is installed
	if err := sh.Run("which", "k6"); err != nil {
		return fmt.Errorf("K6 not found. Install k6 for your platform. https://k6.io/docs/get-started/installation/")
	}

	return nil
}

// Update test and clone repos
func Update() error {
	// tests
	err := doInDir(projectRoot, "tests", func() error {
		if err := sh.RunV("git", "checkout", "main"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}

		if err := sh.RunV("git", "pull"); err != nil {
			return err
		}

		return nil
	})

	// build
	err = doInDir(projectRoot, "build", func() error {
		if err := sh.RunV("git", "checkout", "main"); err != nil {
			return fmt.Errorf("Error checking out grafana test repo %s", err)
		}

		if err := sh.RunV("git", "pull"); err != nil {
			return err
		}

		return nil
	})
	return err
}

// Sets necessary dependencies before real operations
func SetDependencies() error {
	if resolved {
		return nil
	}

	err := ResolveCommit()
	if err != nil {
		return err
	}

	err = ResolveArch()
	if err != nil {
		return err
	}

	err = ResolveINI()
	if err != nil {
		return err
	}

	resolved = true
	return nil
}

// BuildCommit builds a grafana binary and stores it in the artifacts folder
// usage: COMMIT=k8s-proof-of-concept mage buildcommit
func BuildCommit() error {
	err := SetDependencies()
	if err != nil {
		return err
	}

	exists, _ := pathExists(artifact_path)
	if exists {
		fmt.Println("build artifacts cached, skipping build")
		return nil
	}

	// do the build
	err = doInDir(projectRoot, "build", func() error {
		ref := fmt.Sprintf("--grafana-ref=%s", commit)
		distro := fmt.Sprintf("--distro=%s", arch)
		err := sh.RunV("go", "run", "./cmd", "--verbose", ref, "backend", "build", distro)
		return err
	})
	if err != nil {
		return err
	}

	// copy build to artifact path
	// artifacts grafana, grafana-server, grafana-cli
	fmt.Println("copying executable to:", artifact_path)
	buildPath := path.Join(projectRoot, "build", "bin", arch, "grafana")
	if err := sh.RunV("cp", buildPath, artifact_path); err != nil {
		return err
	}

	return nil
}

// TestCommit tests a commit. If you don't set the commit environment variable,
// it will default to main and resolve the git hash. You can also set commit to
// be a branch and it will grab the latest commit for that branch.
// usage: COMMIT=k8s-proof-of-concept mage testcommit
//
// By default we will look for a custom.ini in the project root, however, you
// can also specify this by environment variable and path.
//
// usage: INI=custom.ini mage testcommit
func TestCommit() error {
	var err error

	if err := SetDependencies(); err != nil {
		return err
	}

	// do the build if we need it
	if err := BuildCommit(); err != nil {
		return err
	}

	fmt.Println("setting up work directory")

	// delete old workdir if exists
	if err := sh.RunV("rm", "-rf", path.Join(projectRoot, "work")); err != nil {
		return err
	}

	// copy template directory
	templateConf := path.Join(projectRoot, "templates")
	workConfPath := path.Join(projectRoot, "work")
	if err := sh.RunV("cp", "-r", templateConf, workConfPath); err != nil {
		return err
	}

	// get default.ini for that commit
	iniArtifact := fmt.Sprintf("%s_defaults.ini", commit)
	iniArtifactPath := path.Join(projectRoot, "artifacts", iniArtifact)
	exists, _ := pathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", commit)
		if err := sh.RunV("curl", iniUrl, "-o", iniArtifactPath); err != nil {
			return err
		}
	}

	// copy ini to workdir
	iniWorkPath := path.Join(projectRoot, "work", "conf", "defaults.ini")
	if err := sh.RunV("cp", iniArtifactPath, iniWorkPath); err != nil {
		return err
	}

	// copy custom.ini into work dir
	if iniPath != "" {
		fmt.Println("found custom.ini")
		customIniWorkPath := path.Join(projectRoot, "work", "conf", "custom.ini")
		if err := sh.Run("cp", iniPath, customIniWorkPath); err != nil {
			return err
		}
	}

	// copy artifact
	workExecutable := path.Join(projectRoot, "work", "grafana")
	if err := sh.RunV("cp", artifact_path, workExecutable); err != nil {
		return err
	}

	// boot grafana
	fmt.Println("booting grafana")
	cmd := exec.Command(workExecutable, "server")
	err = doInDir(projectRoot, "work", func() error {
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

	// run k6 tests
	err = doInDir(projectRoot, "tests", func() error {
		if err := sh.RunV("k6", "run", "tests/dashboards.js"); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Do function in a directory
func doInDir(workdir string, operationDir string, fn func() error) error {
	os.Chdir(operationDir)
	defer os.Chdir(workdir)
	return fn()
}

// Checks for existence of directory
func pathExists(path string) (bool, error) {
	// Use the Stat function to check if the directory exists
	_, err := os.Stat(path)

	// If there is no error, the path exists
	if err == nil {
		return true, nil
	}

	// If the error is "not exists", the directory does not exist
	if os.IsNotExist(err) {
		return false, nil
	}

	// If the error is not "not exists", return the error
	return false, err
}

// Get working directory
func getWorkdir() string {
	// Use the Getwd function to get the current working directory
	dir, err := os.Getwd()

	// If there was an error, return it
	if err != nil {
		panic(err)
	}

	// Otherwise, return the current working directory
	return dir
}
