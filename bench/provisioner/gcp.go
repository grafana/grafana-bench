package provisioner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
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
	artifactName := getBundleName(ps.Build.GrafanaRevision, ps.Build.Arch)
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

	// TODO read the test vm instance

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
	return nil
}

// create a package to be uploaded to cache and downloaded on instance
// returns path to bundle
func (d *GCPDriver) prepareBundle(ctx context.Context, ps *ProvisionState) (string, error) {
	_, err := setupGrafanaWorkdir(ctx, d.buildCache, ps)
	if err != nil {
		return "", err
	}

	// TODO start here
	// figure out why archive is the wrong type
	// then continue manually testing booting grafana
	// 1. boot
	// 2. check to see if port is open
	// 3. think about some kind of tag for the VM to mark it to be deleted in 24
	// hours

	// compress the folder
	bundlePath := path.Join(ps.LocalDir, getBundleName(ps.Build.GrafanaRevision, ps.Build.Arch))
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

// generates the name of the bundle
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126_defaults-darwin-arm64-bundle.tar.gz
func getBundleName(grafanaGitRef, arch string) string {
	arch = strings.Replace(arch, "/", "-", -1)
	return fmt.Sprintf("%s-%s-bundle.tar.gz", grafanaGitRef, arch)
}
