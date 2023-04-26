package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

func (b *Config) Bench() error {
	var err error

	if err := b.CheckDeps(); err != nil {
		return err
	}

	if err := b.ResolveConfig(); err != nil {
		return err
	}

	if err := b.ResolveTestSuite(); err != nil {
		return err
	}

	// do the build
	if err := b.Build(); err != nil {
		return err
	}

	// setup workdir
	fmt.Println("setting up work directory")
	executable, err := setupWorkdir(b)
	if err != nil {
		return err
	}

	// boot grafana
	fmt.Println("booting grafana")
	cmd := exec.Command(executable, "server")
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
	liveConfig, err := getLiveConfig()
	if err != nil {
		fmt.Println("error getting live config from booted grafana:", err)
	} else {
		fmt.Println(liveConfig)
	}

	// run k6 tests
	err = utils.DoInDir(b.ProjectRoot, "tests", func() error {
		if err := sh.RunV("k6", "run", "tests/dashboards/dashboard_create.js"); err != nil {
			// k6 run tests/tests/dashboards.js
			return err
		}

		return nil
	})

	return err

}

type LiveConfig struct {
	BuildInfo      map[string]string
	FeatureToggles map[string]string
}

func getLiveConfig() (LiveConfig, error) {
	lc := LiveConfig{}
	url := "http://localhost:3000/api/frontend/settings"
	client := &http.Client{}
	// create new request with event bytes
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(make([]byte, 0)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return lc, err
	}

	// print body
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return lc, err
	}

	err = json.Unmarshal(body, &lc)
	if err != nil {
		return lc, err
	}

	return lc, nil
}

// setupWorkdir sets up directory with configs needed for testing
func setupWorkdir(b *Config) (string, error) {
	// delete old workdir if exists
	if err := sh.RunV("rm", "-rf", path.Join(b.ProjectRoot, "work")); err != nil {
		return "", err
	}

	// copy template directory
	templateConf := path.Join(b.ProjectRoot, "templates")
	workConfPath := path.Join(b.ProjectRoot, "work")
	if err := sh.RunV("cp", "-r", templateConf, workConfPath); err != nil {
		return "", err
	}

	// get default.ini for that commit
	iniArtifact := fmt.Sprintf("%s_defaults.ini", b.GrafanaCommit)
	iniArtifactPath := path.Join(b.ProjectRoot, "artifacts", iniArtifact)
	exists, _ := utils.PathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaCommit)
		if err := sh.RunV("curl", iniUrl, "-o", iniArtifactPath); err != nil {
			return "", err
		}
	}

	// copy ini to workdir
	iniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "defaults.ini")
	if err := sh.RunV("cp", iniArtifactPath, iniWorkPath); err != nil {
		return "", err
	}

	// copy custom.ini into work dir
	if b.GrafanaINIPath != "" {
		fmt.Println("found custom.ini")
		customIniWorkPath := path.Join(b.ProjectRoot, "work", "conf", "custom.ini")
		if err := sh.Run("cp", b.GrafanaINIPath, customIniWorkPath); err != nil {
			return "", err
		}
	}

	// copy artifact
	workExecutable := path.Join(b.ProjectRoot, "work", b.BuildArtifactName)
	if err := sh.RunV("cp", b.BuildArtifactPath, workExecutable); err != nil {
		return "", err
	}
	return workExecutable, nil

}
