package watchers

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-app-sdk/operator"
	"github.com/grafana/grafana-app-sdk/resource"

	"github.com/grafana/grafana-bench/pkg/kubernetes"
)

var _ operator.ResourceWatcher = &BuildJobWatcher{}

type BuildJobWatcher struct {
	PodLogger *kubernetes.PodLogger
	Store     *resource.SimpleStore[kubernetes.JobSpec]
}

func NewBuildJobWatcher(pl *kubernetes.PodLogger, s *resource.SimpleStore[kubernetes.JobSpec]) (*BuildJobWatcher, error) {
	return &BuildJobWatcher{PodLogger: pl, Store: s}, nil
}

func (s *BuildJobWatcher) Add(ctx context.Context, rObj resource.Object) error {
	return nil
}

// Update handles update events for build.Object resources.
func (s *BuildJobWatcher) Update(ctx context.Context, rOld resource.Object, rNew resource.Object) error {
	job, ok := rNew.(*kubernetes.Job)
	if !ok {
		return fmt.Errorf("provided object is not of type *kubernetes.Job(name=%s, namespace=%s, kind=%s)",
			rNew.StaticMetadata().Name, rNew.StaticMetadata().Namespace, rNew.StaticMetadata().Kind)
	}

	oldJob, ok := rOld.(*kubernetes.Job)
	if !ok {
		return fmt.Errorf("provided object is not of type *kubernetes.Job(name=%s, namespace=%s, kind=%s)",
			rNew.StaticMetadata().Name, rOld.StaticMetadata().Namespace, rOld.StaticMetadata().Kind)
	}

	// Log output when job first completes
	if job.Status.Succeeded != oldJob.Status.Succeeded && job.Status.Succeeded == 1 {
		output, err := s.PodLogger.LogsForJob(ctx, job)
		if err != nil {
			return err
		}
		fmt.Println(output)
		err = s.Store.Delete(ctx, job.StaticMeta.Identifier())
		if err != nil {
			return err
		}
		fmt.Println("job removed")
	}

	return nil
}

// Delete handles delete events for build.Object resources.
func (s *BuildJobWatcher) Delete(ctx context.Context, rObj resource.Object) error {
	return nil
}

// Sync is not a standard resource.Watcher function, but is used when wrapping this watcher in an operator.OpinionatedWatcher.
// It handles resources which MAY have been updated during an outage period where the watcher was not able to consume events.
func (s *BuildJobWatcher) Sync(ctx context.Context, rObj resource.Object) error {
	job, ok := rObj.(*kubernetes.Job)
	if !ok {
		return fmt.Errorf("provided object is not of type *kubernetes.Job(name=%s, namespace=%s, kind=%s)",
			rObj.StaticMetadata().Name, rObj.StaticMetadata().Namespace, rObj.StaticMetadata().Kind)
	}

	fmt.Println("Possible update to ", job.StaticMetadata().Identifier())
	return nil
}
