package provisioner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"text/template"
	"time"

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
			return fmt.Errorf("GCPDriver terraform init err: %w", err)
		}

		// terraform apply
		if err := utils.ExecStdout(exec.Command("terraform", "apply", "-auto-approve")); err != nil {
			return fmt.Errorf("GCPDriver terraform apply err: %w", err)
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
	WaitForLiveGrafana(ps.Log, ps.GrafanaInstance.ServiceAddress())
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
		return fmt.Errorf("error running test suite: %w", err)
	}

	// bundle test suite
	testBundleName := getTestBundleName(tr.SuiteRevision)

	// don't cache test suite
	//exists, err := d.buildCache.RemoteExists(ctx, buildcache.TypeTestBundle, testBundleName)
	//if !exists {
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
	//}

	// get presigned url to download bundle
	bundleUrl, err := d.buildCache.GetPresignedUrl(ctx, buildcache.TypeTestBundle, testBundleName)
	if err != nil {
		return err
	}

	ps.Log.Info("got bundle URL", "url", bundleUrl)

	// connect to k6 instance
	ps.Log.Info("connecting to k6 instance")
	connection, err := ps.K6Instance.Connect()
	if err != nil {
		return err
	}
	defer connection.Close()

	// download the test bundle
	ps.Log.Info("downloading test bundle", "url", bundleUrl)
	err = ps.K6Instance.RunCmd(connection, fmt.Sprintf("curl \"%s\" -o /tmp/testbundle.tar.gz", bundleUrl))
	if err != nil {
		return err
	}

	// extract the test suite
	testSuitePath := "/tmp/tests"
	ps.Log.Info("unpacking test bundle", "path", testSuitePath)
	err = ps.K6Instance.RunCmd(connection, fmt.Sprintf("mkdir -p %s && tar -xvf /tmp/testbundle.tar.gz --directory=/tmp/tests", testSuitePath))
	if err != nil {
		return err
	}

	// get tests to run indexed to test suite directory
	ps.Log.Info("getting list of tests to execute")
	tests, err := tr.GetRemoteTestSuiteFiles(testSuitePath)
	if err != nil {
		return err
	}

	ps.Log.Info("getting instance machine spec")
	machineSpec, err := d.GetMachineSpec(ctx, ps)
	if err != nil {
		return err
	}

	// Set environment variables + credentials
	envVars := map[string]string{
		"MACHINE_SPEC":        machineSpec,
		"TEST_SUITE_REVISION": tr.SuiteRevision,
		"GT_URL":              ps.GrafanaInstance.SchemeServiceAddress(),
	}

	if tr.Type == tester.Load {
		envVars["K6_CLOUD_TOKEN"] = tr.K6CloudToken
		envVars["K6_CLOUD_PROJECT_ID"] = tr.K6CloudProjectId
	}

	for _, testFile := range tests {
		ps.Log.Info("running test file", "file", testFile)
		cmd := ""

		if tr.Type == tester.Load {
			cmd = fmt.Sprintf("%s k6 run %s --out cloud", formatEnv(envVars), testFile)
		} else {
			cmd = fmt.Sprintf("%s k6 run %s", formatEnv(envVars), testFile)
		}

		ps.Log.Info("running test on VM", "cmd", cmd)
		err := ps.K6Instance.RunCmd(connection, cmd)
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
	bundlePath := path.Join(ps.LocalDir, getGrafanaBundleName(ps.Build.GrafanaRevision, ps.Build.Arch))
	ps.Log.Info("compressing grafana bundle", "bundlePath", bundlePath)
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
		ExpireDate       string
	}{
		Credentials:      d.credentialsPath,
		Identifier:       ps.Identifier,
		GrafanaBundleUrl: grafanaBundleUrl,
		ExpireDate:       time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

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
