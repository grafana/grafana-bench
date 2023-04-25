package bench

import (
	"fmt"
	"net"
	"os/exec"
	"path"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

func (b *Config) Bench() error {
	var err error

	if err := b.ResolveConfig(); err != nil {
		return err
	}

	// do the build if we need it
	if err := b.Build(); err != nil {
		return err
	}

	fmt.Println("setting up work directory")

	// delete old workdir if exists
	if err := sh.RunV("rm", "-rf", path.Join(b.ProjectRoot, "work")); err != nil {
		return err
	}

	// copy template directory
	templateConf := path.Join(b.ProjectRoot, "templates")
	workConfPath := path.Join(b.ProjectRoot, "work")
	if err := sh.RunV("cp", "-r", templateConf, workConfPath); err != nil {
		return err
	}

	// get default.ini for that commit
	iniArtifact := fmt.Sprintf("%s_defaults.ini", b.GrafanaCommit)
	iniArtifactPath := path.Join(b.ProjectRoot, "artifacts", iniArtifact)
	exists, _ := utils.PathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaCommit)
		if err := sh.RunV("curl", iniUrl, "-o", iniArtifactPath); err != nil {
			return err
		}
	}

	// copy ini to workdir
	iniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "defaults.ini")
	if err := sh.RunV("cp", iniArtifactPath, iniWorkPath); err != nil {
		return err
	}

	// copy custom.ini into work dir
	if b.GrafanaINIPath != "" {
		fmt.Println("found custom.ini")
		customIniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "custom.ini")
		if err := sh.Run("cp", b.GrafanaINIPath, customIniWorkPath); err != nil {
			return err
		}
	}

	// copy artifact
	workExecutable := path.Join(b.ProjectRoot, "work", b.BuildArtifactName)
	if err := sh.RunV("cp", b.BuildArtifactPath, workExecutable); err != nil {
		return err
	}

	// boot grafana
	fmt.Println("booting grafana")
	cmd := exec.Command(workExecutable, "server")
	err = utils.DoInDir(b.ProjectRoot, "work", func() error {
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
	defer func() {
		err := cmd.Process.Kill()
		if err != nil {
			fmt.Println("ERROR killing grafana PID:", err)
		}
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
	err = utils.DoInDir(b.ProjectRoot, "tests", func() error {
		if err := sh.RunV("k6", "run", "tests/dashboards.js"); err != nil {
			// k6 run tests/tests/dashboards.js
			return err
		}

		return nil
	})

	return err

}
