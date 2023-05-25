package kubernetes

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-app-sdk/resource"
)

func JobSchema() resource.Schema {
	return resource.NewSimpleSchema("batch", "v1", &Job{}, resource.WithKind("Job"))
}

type Job struct {
	resource.BasicMetadataObject
	Spec   JobSpec   `json:"spec"`
	Status JobStatus `json:"status"`
}

func (j *Job) Copy() resource.Object {
	return resource.CopyObject(j)
}

func (j *Job) Unmarshal(obj resource.ObjectBytes, config resource.UnmarshalConfig) error {
	if config.WireFormat != resource.WireFormatJSON {
		return fmt.Errorf("unsupported wire format: %v", config.WireFormat)
	}

	err := json.Unmarshal(obj.Spec, &j.Spec)
	if err != nil {
		return err
	}

	err = json.Unmarshal(obj.Metadata, &j.BasicMetadataObject.CommonMeta)
	if err != nil {
		return err
	}

	err = json.Unmarshal(obj.Subresources["status"], &j.Status)
	if err != nil {
		return err
	}

	return nil
}

func (j *Job) SpecObject() any {
	return j.Spec
}

func (j *Job) Subresources() map[string]any {
	return map[string]any{"status": j.Status}
}

type JobStatus struct {
	CompletionTime string
	Succeeded      int
}

// This nesting produces kubernetes readable objects for executing jobs. The
// Json naming of the fields is specific to kubernetes and should not be
// changed. In general you should use NewJob
// yaml ex:
//
//	spec:
//	  template:
//	  spec:
//	    containers:
//	    - name: pi
//	      image: perl:5.34.0
//	      command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
//	    restartPolicy: Never
//	backoffLimit: 4
type JobSpec struct {
	Template     JobTemplate `json:"template"`
	BackoffLimit int         `json:"backoffLimit"`
}

type JobTemplate struct {
	Spec JobDefinition `json:"spec"`
}

type JobDefinition struct {
	Containers    []ContainerDefinition `json:"containers"`
	RestartPolicy string                `json:"restartPolicy"`
}

type ContainerDefinition struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command"`
}

// Creates a new jobspec in the correct kubernetes format
func NewJob(name, image string, command []string) JobSpec {
	return JobSpec{
		BackoffLimit: 1,
		Template: JobTemplate{
			Spec: JobDefinition{
				RestartPolicy: "Never",
				Containers: []ContainerDefinition{
					{
						Name:    name,
						Image:   image,
						Command: command,
					},
				},
			},
		},
	}
}
