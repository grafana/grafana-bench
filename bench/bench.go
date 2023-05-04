package bench

import (
	"context"
	"fmt"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

func (b *Config) Bench(ctx context.Context) error {
	var err error

	if err := b.ResolveConfig(ctx); err != nil {
		return err
	}

	if err := b.Build(ctx); err != nil {
		return err
	}

	// setup workdir
	fmt.Println("setting up work directory")
	executable, err := b.setupWorkdir(ctx)
	if err != nil {
		return err
	}

	// boot grafana
	fmt.Println("booting grafana")
	killFunc, err := b.Boot(ctx, executable)
	if err != nil {
		return err
	}
	defer killFunc()

	// run k6 tests
	return b.Test(ctx)
}

// setupWorkdir sets up directory with configs needed for testing a grafana
// build
func (b *Config) setupWorkdir(ctx context.Context) (string, error) {
	// verify executable exists

	// TODO START HERE
	// 1. test logic for checking build cache
	// 2. think through workflow of when build is called. Maybe that should just
	// happen in this method as we're getting ready?
	// 3. start working on writing builds to buildcache if they don't already
	// exist there
	// 4. add lifecycle policy to buildcache prefix

	exists, _ := utils.PathExists(b.BuildArtifactPath)
	if !exists {
		// no build cache configured
		if b.RemoteBuildCache == nil {
			return "", fmt.Errorf(
				"Build artifact not found on disk or in cache: %s. GRAFANA_REVISION=commit:%s mage build first",
				b.BuildArtifactPath,
				b.GrafanaRevision,
			)
		}

		// attempt to get it from the build cache
		obj := b.RemoteBuildCache.GetObjectHandleFromGrafanaBuildName(b.BuildArtifactName)
		fmt.Println("build-cache: checking build cache for artifact:", obj.ObjectName())

		// check to see if it exists
		exists, err := buildcache.ObjectExists(ctx, obj)
		if err != nil {
			return "", fmt.Errorf("Error contacting build cache: %s", err)
		}
		if !exists {
			return "", fmt.Errorf("Object does not exist: %s", obj.ObjectName())
		}

		// if exists, download
		fmt.Println("build-cache: object found, downloading")
		if err := buildcache.WriteToLocal(ctx, b.BuildArtifactPath, obj); err != nil {
			return "", fmt.Errorf("Error retrieving build artifact from bucket:", err)
		}
	}

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
	iniArtifact := fmt.Sprintf("%s_defaults.ini", b.GrafanaRevision)
	iniArtifactPath := path.Join(b.ProjectRoot, "artifacts", iniArtifact)
	exists, _ = utils.PathExists(iniArtifactPath)
	if !exists {
		// get the ini for that commit of grafana if it doesn't exist
		// takes 7 chars to full commit hash
		iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", b.GrafanaRevision)
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

//type LiveConfig struct {
//  BuildInfo      map[string]string
//  FeatureToggles map[string]string
//}

//func getLiveConfig() (LiveConfig, error) {
//  lc := LiveConfig{}
//  url := "http://localhost:3000/api/frontend/settings"
//  client := &http.Client{}
//  // create new request with event bytes
//  req, err := http.NewRequest("GET", url, bytes.NewBuffer(make([]byte, 0)))
//  if err != nil {
//    panic(err)
//  }

//  req.Header.Set("Content-Type", "application/json")

//  resp, err := client.Do(req)
//  if err != nil {
//    return lc, err
//  }

//  // print body
//  defer func() {
//    _ = resp.Body.Close()
//  }()

//  body, err := io.ReadAll(resp.Body)
//  if err != nil {
//    return lc, err
//  }

//  err = json.Unmarshal(body, &lc)
//  if err != nil {
//    return lc, err
//  }

//  return lc, nil
//}
