package provisioner

import (
	"context"
	"html/template"
	"os"
	"os/exec"
	"path"

	"github.com/grafana/grafana-bench/bench/buildcache"
	"github.com/grafana/grafana-bench/bench/tester"
	"github.com/grafana/grafana-bench/bench/utils"
)

// TODO don't hardcode credentials
var credentials string = "/Users/jeff/projects/g/bench/GCP-infra-manager-828bbfa6f427.json"

var _ ProvisionDriver = (*GCPDriver)(nil)

type GCPDriver struct {
	terraformTemplates map[string]*template.Template
	buildCache         *buildcache.BuildCache
}

func NewGCPDriver(buildCache *buildcache.BuildCache, terraformTemplates map[string]*template.Template) *GCPDriver {
	return &GCPDriver{
		terraformTemplates: terraformTemplates,
		buildCache:         buildCache,
	}
}

func (d *GCPDriver) Provision(ctx context.Context, ps *ProvisionState) (func() error, error) {

	// TODO START HERE
	// figure out what building the directory and shipping to build cache looks
	// like
	// 1. what should we name the bundle?
	// 2. what should the artifact type be?
	// 3. should we request a presigned url separately or return in this function
	presignedUrl, err := d.prepareBundle(ctx, ps)
	if err != nil {
		return NilFunc, err
	}

	err = d.writeTemplates(ctx, ps, presignedUrl)
	if err != nil {
		return NilFunc, err
	}

	err = utils.DoInDir(utils.Getwd(), ps.StateDir, func() error {
		// terraform init
		if err := utils.ExecStdout(exec.Command("terraform", "init")); err != nil {
			return err
		}

		// terraform apply
		if err := utils.ExecStdout(exec.Command("terraform", "apply", "-auto-approve")); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return NilFunc, err
	}

	// tar the result
	//archiveFilepath := path.Join(v.RootDir, v.Identifier+".tar.gz")

	//cmd := exec.Command("tar", "-czvf", archiveFilepath, "--exclude='.terraform'", path.Join(v.Workdir, "*"))
	//return utils.ExecStdout(cmd)

	// upload to gcs bucket

	// get the script to run
	// run terraform apply

	// what should killFunc do?

	return NilFunc, nil
}

// Blocking call that waits for grafana to become ready
func (d *GCPDriver) WaitForReady(ctx context.Context, ps *ProvisionState) {
	WaitForLiveGrafana(ps.GrafanaAddress)
}

// Check - checks if Grafana + test runner are ready
func (d *GCPDriver) Ready(ctx context.Context, ps *ProvisionState) bool {
	return IsLive(ps.GrafanaAddress)
}

// Destroy - destroys a provisioned instance of Grafana + test runner
func (d *GCPDriver) Destroy(ctx context.Context, ps *ProvisionState) error {
	return nil
}

func (d *GCPDriver) RunTests(ctx context.Context, ps *ProvisionState, tr *tester.TestRun) error {
	return nil
}

// create a package to be uploaded to the instance
func (d *GCPDriver) prepareBundle(ctx context.Context, ps *ProvisionState) (string, error) {
	return "", nil
}

// writes state files to disk
func (d *GCPDriver) writeTemplates(ctx context.Context, ps *ProvisionState, presignedUrl string) error {
	// write terraform template
	terraformPlanFile, err := os.Create(path.Join(ps.StateDir, "main.tf"))
	if err != nil {
		return err
	}
	defer terraformPlanFile.Close()

	err = d.terraformTemplates["gcp_basic.tmpl"].Execute(terraformPlanFile, ps)
	if err != nil {
		return err
	}

	// write startup script
	startupScriptFile, err := os.Create(path.Join(ps.StateDir, "startup.sh"))
	if err != nil {
		return err
	}

	return d.terraformTemplates["gcp_startup.sh.tmpl"].Execute(startupScriptFile, ps)

}
