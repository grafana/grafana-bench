package provisioner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"text/template"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

var _ ProvisionDriver = (*GCPDriver)(nil)

type GCPDriver struct {
	terraformTemplates map[string]*template.Template
	buildCache         *buildcache.BuildCache
	credentialsPath    string
}

func NewGCPDriver(buildCache *buildcache.BuildCache, terraformTemplates map[string]*template.Template, credentialsPath string) *GCPDriver {
	return &GCPDriver{
		terraformTemplates: terraformTemplates,
		buildCache:         buildCache,
		credentialsPath:    credentialsPath,
	}
}

func (d *GCPDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {

	// prepare bundle
	bundlePath, err := d.prepareBundle(ctx, ps)
	if err != nil {
		return NilFunc, err
	}

	// cache the bundle
	artifactName := getGrafanaBundleName(ps.Build.GrafanaRevision, ps.Build.Arch)
	err = d.buildCache.StoreFile(ctx, buildcache.TypeBundle, bundlePath, artifactName)
	if err != nil {
		return NilFunc, err
	}

	// get presigned url for bundle
	presignedUrl, err := d.buildCache.GetPresignedUrl(ctx, buildcache.TypeBundle, artifactName)
	if err != nil {
		return NilFunc, err
	}

	// write templates to state dir
	// what struct do I actually need here?
	err = d.writeTemplates(ctx, ps, presignedUrl)
	if err != nil {
		return NilFunc, err
	}

	err = utils.DoInDir(utils.Getwd(), ps.StateDir, func() error {
		// terraform init
		if err := utils.ExecStdout(exec.Command("terraform", "init")); err != nil {
			return fmt.Errorf("provisioner: GCPDriver terraform init err: %w", err)
		}

		// terraform apply
		if err := utils.ExecStdout(exec.Command("terraform", "apply", "-auto-approve")); err != nil {
			return fmt.Errorf("provisioner: GCPDriver terraform apply err: %w", err)
		}

		return nil
	})

	if err != nil {
		return NilFunc, err
	}

	// read grafana instance from the state dir
	ps.GrafanaInstance, err = readVM(ps.StateDir, ps.Identifier, "grafana")
	if err != nil {
		return NilFunc, err
	}
	// TODO don't hardcode this
	ps.GrafanaInstance.ServicePort = "3000"

	// read k6 instance from the state dir
	//ps.K6Instance, err = readVM(ps.StateDir, ps.Identifier, "k6")
	//if err != nil {
	//  return NilFunc, err
	//}

	return NilFunc, nil
}

// Blocking call that waits for grafana to become ready
func (d *GCPDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaInstance.ServiceAddress())
}

// Check - checks if Grafana + test runner are ready
func (d *GCPDriver) Ready(ctx context.Context, ps *ProvisionState) bool {
	return IsLive(ps.GrafanaInstance.ServiceAddress())
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (d *GCPDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	err := utils.DoInDir(utils.Getwd(), ps.StateDir, func() error {
		// terraform init
		if err := utils.ExecStdout(exec.Command("terraform", "init")); err != nil {
			return err
		}

		// terraform destroy
		if err := utils.ExecStdout(exec.Command("terraform", "destroy")); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// remove the local dir and all state and workdir
	return utils.Rm(ps.LocalDir)
}

func (d *GCPDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {

	// resolve test suite to correct version etc
	err := tr.ResolveTestSuite()
	if err != nil {
		return fmt.Errorf("provisioner: error running test suite: %w", err)
	}

	testBundleName := getTestBundleName(tr.SuiteRevision)

	// bundle test suite
	testBundlePath := path.Join(ps.LocalDir, testBundleName)
	err = tr.PrepareTestBundle(testBundlePath)
	if err != nil {
		return err
	}

	// ship to buildcache
	err = d.buildCache.StoreFile(ctx, buildcache.TypeTestBundle, testBundlePath, testBundleName)
	if err != nil {
		return err
	}

	bundleUrl, err := d.buildCache.GetPresignedUrl(ctx, buildcache.TypeTestBundle, testBundleName)
	if err != nil {
		return err
	}

	// connect to machine
	connection, err := ps.K6Instance.Connect()
	if err != nil {
		return err
	}
	defer connection.Close()

	// download the test bundle
	err = ps.K6Instance.Run(fmt.Sprintf("curl %s -o /tmp/testbundle.tar.gz", bundleUrl))
	if err != nil {
		return err
	}

	// extract the test suite
	err = ps.K6Instance.Run(connection, fmt.Sprintf("mkdir -p /tmp/tests", "tar -xvf /tmp/testbundle.tar.gz --directory=/tmp/tests"))
	if err != nil {
		return err
	}

	// START HERE
	// 1. FIGURE OUT HOW TO RUN K6 TESTS
	// 2. SETUP GRAFANA VM TEMPLATE

	// run k6 tests
	err = utils.DoInDir(utils.Getwd(), tr.TestSuiteDir, func() error {
		resultsDir := tr.ResultsDirectory(ps.Identifier)
		err := os.MkdirAll(resultsDir, 0755)
		if err != nil {
			return err
		}

		machineSpec, err := d.GetMachineSpec(ctx, ps)
		if err != nil {
			return err
		}

		envVars := map[string]string{
			// TODO fix this and get the machine spec from the provisioner
			"MACHINE_SPEC":        machineSpec,
			"TEST_SUITE_REVISION": tr.SuiteRevision,
			"TEST_SUMMARY_DIR":    resultsDir,
			"GT_URL":              ps.GrafanaInstance.HttpsServiceAddress(),
		}

		if tr.ReportToK6Cloud {
			envVars["k6_CLOUD_TOKEN"] = tr.K6CloudToken
			//envVars["k6_CLOUD_PROJECT_ID"] = tr.K6CloudProjectID
		}

		tests, err := tr.GetTestSuiteFiles()
		if err != nil {
			return err
		}

		// run the tests
		for _, testFile := range tests {
			fmt.Println("provisioner: running test file:", testFile)

			var cmd *exec.Cmd
			if tr.ReportToK6Cloud {
				cmd = exec.Command("k6", "run", testFile, "-i", "1", "-u", "1", "-o", "cloud")
			} else {
				cmd = exec.Command("k6", "run", testFile, "-i", "1", "-u", "1")
			}

			// TODO figure out what to do with threshold errors from k6.
			// The ones in the test may not match what we need and will exist with
			// non-zero status code resulting in RunWithVar returning an error
			// an error even though we don't care about it. This isn't a GREAT
			// approach. We should figure out a way to tell k6 not to return an error
			// if threshold is breached rather than necessarily modifying the test

			// k6 run tests/tests/dashboards.js -i 1 -u 1 -o cloud
			_ = utils.ExecStdoutWithEnv(cmd, envVars)
		}

		return nil
	})

	return err
}

// create a package to be uploaded to cache and downloaded on instance
// returns path to bundle
func (d *GCPDriver) prepareBundle(ctx context.Context, ps *ProvisionState) (string, error) {
	_, err := setupGrafanaWorkdir(ctx, d.buildCache, ps)
	if err != nil {
		return "", err
	}

	// compress the folder
	bundlePath := path.Join(ps.LocalDir, getGrafanaBundleName(ps.Build.GrafanaRevision, ps.Build.Arch))
	err = utils.CompressFolder(ps.WorkDir, bundlePath)
	if err != nil {
		return "", err
	}

	return bundlePath, nil
}

// writes state files to disk
func (d *GCPDriver) writeTemplates(ctx context.Context, ps *ProvisionState, grafanaBundleUrl string) error {

	templateData := struct {
		Credentials      string
		Identifier       string
		GrafanaBundleUrl string
		GrafanaBinary    string
	}{
		Credentials:      d.credentialsPath,
		Identifier:       ps.Identifier,
		GrafanaBundleUrl: grafanaBundleUrl,
		//TestBundleUrl:    testBundleUrl,
	}

	// TODO
	// Add tag to the terraform template so that we can mark for delete in 24
	// hours assuming it's not already nuked

	// write terraform template
	terraformPlanFile, err := os.Create(path.Join(ps.StateDir, "main.tf"))
	if err != nil {
		return err
	}
	defer terraformPlanFile.Close()

	err = d.terraformTemplates["gcp_basic.tf.tmpl"].Execute(terraformPlanFile, templateData)
	if err != nil {
		return err
	}

	// write startup script
	startupScriptFile, err := os.Create(path.Join(ps.StateDir, "startup.sh"))
	if err != nil {
		return err
	}

	return d.terraformTemplates["grafana_startup.sh.tmpl"].Execute(startupScriptFile, templateData)
}

// Gets machine spec for provisioned machine
// TODO IMPLEMENT ME
func (d *GCPDriver) GetMachineSpec(ctx context.Context, ps *ProvisionState) (string, error) {
	// driver, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|m1max|65536|1|3.2 GHz|amd64|linux", nil
}
