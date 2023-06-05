package main

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"

	"github.com/google/uuid"
	"github.com/grafana/grafana-bench/bench/utils"
)

// FIXME this is hardcoded
var credentials string = "/Users/jeff/projects/g/bench/GCP-infra-manager-828bbfa6f427.json"

var tfTmpl *template.Template

type VM struct {
	RootDir       string
	Credentials   string
	Identifier    string
	Workdir       string
	TerraformFile string
}

func main() {
	var err error

	// parse template
	tmplFile := path.Join("vm", "terraform.tmpl")
	tfTmpl, err = template.ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}

	// create identifier
	uuid := uuid.New()
	fmt.Println("identifier", uuid.String())

	vm := VM{
		Identifier:  uuid.String(),
		RootDir:     utils.GetWorkdir(),
		Credentials: credentials,
	}

	// start here
	if err = vm.setupTerraformDir(); err != nil {
		panic(err)
	}

	if err = vm.provision(); err != nil {
		panic(err)
	}

	if err = vm.cleanupTerraformDir(); err != nil {
		panic(err)
	}

	// make sure we can ping
	if err = vm.ping(); err != nil {
		panic(err)
	}

	if err = vm.gatherState(); err != nil {
		panic(err)
	}
}

func (v *VM) setupTerraformDir() error {
	// create directory
	tempDir, err := os.MkdirTemp("", v.Identifier)
	if err != nil {
		panic(err)
	}
	v.Workdir = tempDir
	fmt.Println("workdir:", v.Workdir)

	// create terraform file
	v.TerraformFile = path.Join(v.Workdir, "main.tf")
	outputFile, err := os.Create(v.TerraformFile)
	if err != nil {
		return fmt.Errorf("Error creating output file: %w", err)
	}
	defer outputFile.Close()

	// copy startup script
	err = utils.CopyFile(
		path.Join("vm", "startup.sh"),
		path.Join(v.Workdir, "startup.sh"),
	)
	if err != nil {
		return err
	}

	// write template into directory
	err = tfTmpl.Execute(outputFile, v)
	if err != nil {
		panic(err)
	}

	return nil
}

func (v *VM) provision() error {
	err := utils.DoInDir(v.RootDir, v.Workdir, func() error {
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
		return err
	}

	// tar the result
	archiveFilepath := path.Join(v.RootDir, v.Identifier+".tar.gz")

	cmd := exec.Command("tar", "-czvf", archiveFilepath, "--exclude='.terraform'", path.Join(v.Workdir, "*"))
	return utils.ExecStdout(cmd)

	// upload to gcs bucket
}

func (v *VM) destroy() error {
	return utils.DoInDir(v.RootDir, v.Workdir, func() error {
		return utils.ExecStdout(exec.Command("terraform", "init"))
	})
}

func (v *VM) cleanupTerraformDir() error {
	// ensure terraform state before delete
	// rmdir
	return nil
}

// Blocking call until we get successful ping resp back
func (v *VM) ping() error {
	return nil
}

// Retrieves terraform state + credentials from GCS bucket and puts them in
// local directory
func (v *VM) gatherState() error {
	// put this in new tempfile for testing
	return nil
}
