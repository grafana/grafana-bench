package main

import (
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/google/uuid"
)

var tfTmpl *template.Template

type VM struct {
	Credentials   string
	Identifier    string
	Workdir       string
	TerraformFile string
}

func main() {
	var err error

	// parse template
	tmplFile := "vm/terraform.tmpl"
	tfTmpl, err = template.ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}

	// create identifier
	uuid := uuid.New()
	fmt.Println("identifier", uuid.String())

	vm := VM{
		Identifier: uuid.String(),
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

	// set global path to credentials
	// TODO don't hardcode this
	v.Credentials = path.Join("/Users/jeff/projects/g/bench/GCP-infra-manager-828bbfa6f427.json")

	// write template into directory
	err = tfTmpl.Execute(outputFile, v)
	if err != nil {
		panic(err)
	}

	return nil
}

func (v *VM) provision() error {
	// terraform init
	// terraform apply
	// upload to gcs bucket
	return nil
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
