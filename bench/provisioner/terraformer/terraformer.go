package terraformer

import (
	"context"
	"html/template"
	"log"
	"os"
	"path"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// Terraformer is responsible for provisioning and deprovisioning VM's as well as storing the state of runs
type Terraformer struct {
	credPath          string
	LocalDir          string
	Client            *storage.Client
	Bucket            *storage.BucketHandle
	TerraformTemplate *template.Template
	// Use a remote bucket to cache builds
	RemoteCache bool
	VmEnabled   bool
}

// State contains all the data relevant to manage VM's
type State struct {
	StateDir      string
	Identifier    string
	TerraformFile string
	Initialized   bool
}

func NewTerraformer(ctx context.Context, localDir, credPath, bucketName, tfTemplatePath string) (*Terraformer, error) {

	terraformer := &Terraformer{
		LocalDir:    localDir,
		RemoteCache: false, // default
	}

	// create dir
	log.Println("terraformer: using local directory:", localDir)
	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		return nil, err
	}

	// ignore setup if no remote cache
	if bucketName == "" {
		log.Println("build-cache: no remote store defined")
		return terraformer, nil
	}

	log.Println("terraformer: using template:", tfTemplatePath)
	terraformer.TerraformTemplate, err = template.ParseFiles(tfTemplatePath)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, err
	}

	return &Terraformer{
		Client:      client,
		Bucket:      client.Bucket(bucketName),
		credPath:    credPath,
		LocalDir:    localDir,
		RemoteCache: true,
	}, nil
}

// Creates a new state with passed in configuration. Call initialize to
// initiatlize the local state
func (t *Terraformer) NewState() *State {
	// create identifier
	uuid := uuid.Must(uuid.NewRandom())
	log.Println("terraformer: new state identifier:", uuid.String())

	stateDir := path.Join(t.LocalDir, uuid.String())
	log.Println("terraformer: directory initialized:", stateDir)

	return &State{
		Identifier: uuid.String(),
		StateDir:   stateDir,
	}
}

func (s *State) Init() error {
	// Create the directory
	err := os.MkdirAll(s.StateDir, 0755)
	if err != nil {
		return err
	}

	s.Initialized = true

	return nil
}

// Initialize a state

// List states
// Create states
