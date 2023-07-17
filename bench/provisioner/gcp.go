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
	ps.GrafanaInstance, err = readVM(ps.StateDir, "grafana")
	if err != nil {
		return NilFunc, err
	}

	// read k6 instance from the state dir
	ps.K6Instance, err = readVM(ps.StateDir, "k6")
	if err != nil {
		return NilFunc, err
	}

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

// Destroy - destroys a provisioned instance of Grafana and K6 test runner
// removing all state after
func (d *GCPDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	err := utils.DoInDir(utils.Getwd(), ps.StateDir, func() error {
		// terraform init
		if err := utils.ExecStdout(exec.Command("terraform", "init")); err != nil {
			return err
		}

		// terraform destroy
		if err := utils.ExecStdout(exec.Command("terraform", "destroy", "-auto-approve")); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// TODO think about what we might need to clean from the build cache

	// remove the local dir and all state and workdir
	return utils.Rm(ps.LocalDir)
}

func (d *GCPDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {

	// resolve test suite to correct version etc
	err := tr.ResolveTestSuite()
	if err != nil {
		return fmt.Errorf("provisioner: error running test suite: %w", err)
	}

	// bundle test suite
	testBundleName := getTestBundleName(tr.SuiteRevision)
	exists, err := d.buildCache.RemoteExists(ctx, buildcache.TypeTestBundle, testBundleName)
	if !exists {
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
	}

	// get presigned url to download bundle
	bundleUrl, err := d.buildCache.GetPresignedUrl(ctx, buildcache.TypeTestBundle, testBundleName)
	if err != nil {
		return err
	}

	fmt.Println("provisioner: bundleurl:", bundleUrl)

	// connect to k6 instance
	fmt.Println("provisioner: connecting to k6 instance")
	connection, err := ps.K6Instance.Connect()
	if err != nil {
		return err
	}
	defer connection.Close()

	// download the test bundle
	fmt.Println("provisioner: downloading test bundle")
	err = ps.K6Instance.Run(connection, fmt.Sprintf("curl \"%s\" -o /tmp/testbundle.tar.gz", bundleUrl))
	if err != nil {
		return err
	}

	// extract the test suite
	testSuitePath := "/tmp/tests"
	fmt.Println("provisioner: unpacking test bundle")
	err = ps.K6Instance.Run(connection, fmt.Sprintf("mkdir -p %s && tar -xvf /tmp/testbundle.tar.gz --directory=/tmp/tests", testSuitePath))
	if err != nil {
		return err
	}

	// get tests to run indexed to test suite directory
	fmt.Println("provisioner: getting list of tests to execute")
	tests, err := tr.GetRemoteTestSuiteFiles(testSuitePath)
	if err != nil {
		return err
	}

	// create test results dir
	resultsDir := "/tmp/results"
	fmt.Println("provisioner: creating test result dir:", resultsDir)
	err = ps.K6Instance.Run(connection, fmt.Sprintf("mkdir -p %s", resultsDir))
	if err != nil {
		return err
	}

	fmt.Println("provisioner: getting instance machine spec")
	machineSpec, err := d.GetMachineSpec(ctx, ps)
	if err != nil {
		return err
	}

	// Set environment variables + credentials
	envVars := map[string]string{
		"MACHINE_SPEC":        machineSpec,
		"TEST_SUITE_REVISION": tr.SuiteRevision,
		"TEST_SUMMARY_DIR":    resultsDir,
		"GT_URL":              ps.GrafanaInstance.HttpServiceAddress(),
	}

	if tr.ReportToK6Cloud {
		envVars["K6_CLOUD_TOKEN"] = tr.K6CloudToken
		envVars["K6_CLOUD_PROJECT_ID"] = "3641403"
	}

	for _, testFile := range tests {
		fmt.Println("provisioner: running test file:", testFile)
		cmd := fmt.Sprintf("%s k6 run %s --out cloud", formatEnv(envVars), testFile)
		//cmd := fmt.Sprintf("%s k6 run %s -i 1 -u 1 --out cloud", formatEnv(envVars), testFile)
		fmt.Println(cmd)
		err := ps.K6Instance.Run(connection, cmd)
		if err != nil {
			return err
		}
	}

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
	fmt.Println("provisioner: compressing grafana bundle")
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

	// write grafana startup script
	grafanaStartupScriptFile, err := os.Create(path.Join(ps.StateDir, "grafana_startup.sh"))
	if err != nil {
		return err
	}
	err = d.terraformTemplates["grafana_startup.sh.tmpl"].Execute(grafanaStartupScriptFile, templateData)
	if err != nil {
		return err
	}

	// write k6 startup script
	k6StartupScriptFile, err := os.Create(path.Join(ps.StateDir, "k6_startup.sh"))
	if err != nil {
		return err
	}
	err = d.terraformTemplates["k6_startup.sh.tmpl"].Execute(k6StartupScriptFile, templateData)

	return err
}

// Gets machine spec for provisioned machine
// TODO IMPLEMENT ME
func (d *GCPDriver) GetMachineSpec(ctx context.Context, ps *ProvisionState) (string, error) {
	// driver, process/machine, memory, # cores, clockspeed, architecture, os
	return "local|m1max|65536|1|3.2 GHz|amd64|linux", nil
}
