package watchers

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-app-sdk/operator"
	"github.com/grafana/grafana-app-sdk/resource"

	"github.com/grafana/grafana-bench/pkg/generated/resource/build"
	"github.com/grafana/grafana-bench/pkg/kubernetes"
)

var _ operator.ResourceWatcher = &BuildWatcher{}

type BuildWatcher struct {
	store *resource.SimpleStore[kubernetes.JobSpec]
}

func NewBuildWatcher(store *resource.SimpleStore[kubernetes.JobSpec]) (*BuildWatcher, error) {
	return &BuildWatcher{
		store: store,
	}, nil
}

// Add handles add events for build.Object resources.
func (s *BuildWatcher) Add(ctx context.Context, rObj resource.Object) error {

	// check to see if thing exists in bucket
	// if it does, change status to completed

	object, ok := rObj.(*build.Object)
	if !ok {
		return fmt.Errorf("provided object is not of type *build.Object (name=%s, namespace=%s, kind=%s)",
			rObj.StaticMetadata().Name, rObj.StaticMetadata().Namespace, rObj.StaticMetadata().Kind)
	}

	fmt.Println("Added ", object.StaticMetadata().Identifier())

	// create the build job
	buildName := getJobName(object.Spec)
	resourceIdentifier := resource.Identifier{Name: buildName, Namespace: "default"}

	cmd := []string{"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"}
	jobSpec := kubernetes.NewJob(buildName, "perl:5.34.0", cmd)
	_, err := s.store.Add(ctx, resourceIdentifier, jobSpec)
	if err != nil {
		return err
	}

	// TODO
	return nil
}

func getJobName(buildSpec build.Spec) string {
	shortRevision := buildSpec.GitRevision[:7]
	return fmt.Sprintf("build-%s-%s-%s-%s", buildSpec.ApplicationName, shortRevision, buildSpec.Architecture, buildSpec.OperatingSystem)
}

// Update handles update events for build.Object resources.
func (s *BuildWatcher) Update(ctx context.Context, rOld resource.Object, rNew resource.Object) error {
	oldObject, ok := rOld.(*build.Object)
	if !ok {
		return fmt.Errorf("provided object is not of type *build.Object (name=%s, namespace=%s, kind=%s)",
			rOld.StaticMetadata().Name, rOld.StaticMetadata().Namespace, rOld.StaticMetadata().Kind)
	}

	newObject, ok := rNew.(*build.Object)
	if !ok {
		return fmt.Errorf("provided object is not of type *build.Object (name=%s, namespace=%s, kind=%s)",
			rNew.StaticMetadata().Name, rNew.StaticMetadata().Namespace, rNew.StaticMetadata().Kind)
	}

	// TODO
	fmt.Println("Updated ", oldObject.StaticMetadata().Identifier(), newObject.StaticMetadata().Identifier())
	return nil
}

// Delete handles delete events for build.Object resources.
func (s *BuildWatcher) Delete(ctx context.Context, rObj resource.Object) error {
	object, ok := rObj.(*build.Object)
	if !ok {
		return fmt.Errorf("provided object is not of type *build.Object (name=%s, namespace=%s, kind=%s)",
			rObj.StaticMetadata().Name, rObj.StaticMetadata().Namespace, rObj.StaticMetadata().Kind)
	}

	// TODO
	fmt.Println("Deleted ", object.StaticMetadata().Identifier())
	return nil
}

// Sync is not a standard resource.Watcher function, but is used when wrapping this watcher in an operator.OpinionatedWatcher.
// It handles resources which MAY have been updated during an outage period where the watcher was not able to consume events.
func (s *BuildWatcher) Sync(ctx context.Context, rObj resource.Object) error {
	object, ok := rObj.(*build.Object)
	if !ok {
		return fmt.Errorf("provided object is not of type *build.Object (name=%s, namespace=%s, kind=%s)",
			rObj.StaticMetadata().Name, rObj.StaticMetadata().Namespace, rObj.StaticMetadata().Kind)
	}

	// TODO
	fmt.Println("Possible update to ", object.StaticMetadata().Identifier())
	return nil
}
