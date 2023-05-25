package main

import (
	"fmt"
	"github.com/grafana/grafana-app-sdk/plugin/kubeconfig"
	"os"
	"os/signal"
	"syscall"

	"github.com/grafana/grafana-bench/pkg/generated/resource/build"
	"github.com/grafana/grafana-bench/pkg/kubernetes"
	"github.com/grafana/grafana-bench/pkg/watchers"

	"github.com/grafana/grafana-app-sdk/k8s"
	"github.com/grafana/grafana-app-sdk/operator"
	"github.com/grafana/grafana-app-sdk/resource"
)

func main() {
	// Load the kube config
	kubeConfig, err := LoadInClusterConfig()
	if err != nil {
		panic(err)
	}

	// Create our client generator, using kubernetes as a store
	clientGenerator := k8s.NewClientRegistry(kubeConfig.RestConfig, k8s.ClientConfig{})

	// Create the controller which we'll attach our informer(s) and watcher(s) to
	controller := operator.NewInformerController()
	controller.ErrorHandler = func(err error) {
		fmt.Println(err)
	}

	// Wrap our resource watchers in OpinionatedWatchers, then add them to the controller
	buildClient, err := clientGenerator.ClientFor(build.Schema())
	if err != nil {
		panic(err)
	}

	jobStore, err := resource.NewSimpleStore[kubernetes.JobSpec](kubernetes.JobSchema(), clientGenerator)
	if err != nil {
		panic(err)
	}

	buildWatcher, err := watchers.NewBuildWatcher(jobStore)
	if err != nil {
		panic(err)
	}
	buildOpinionatedWatcher, err := operator.NewOpinionatedWatcher(build.Schema(), buildClient)
	if err != nil {
		panic(err)
	}
	buildOpinionatedWatcher.Wrap(buildWatcher, false)
	buildOpinionatedWatcher.SyncFunc = buildWatcher.Sync
	err = controller.AddWatcher(buildOpinionatedWatcher, build.Schema().Kind())
	if err != nil {
		panic(err)
	}

	// Add informers for each of our resource types
	buildInformer, err := operator.NewKubernetesBasedInformer(build.Schema(), buildClient, kubeConfig.Namespace)
	if err != nil {
		panic(err)
	}
	err = controller.AddInformer(buildInformer, build.Schema().Kind())
	if err != nil {
		panic(err)
	}

	// register job watcher
	err = RegisterJobWatcher(kubeConfig, controller, clientGenerator, jobStore)
	if err != nil {
		panic(err)
	}

	// Create our operator
	op := operator.New()
	op.AddController(controller)

	stopCh := make(chan struct{})

	// Signal channel
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		stopCh <- struct{}{}
	}()

	// Run
	op.Run(stopCh)
}

func RegisterJobWatcher(kubeConfig *kubeconfig.NamespacedConfig, controller *operator.InformerController, clientGenerator resource.ClientGenerator, store *resource.SimpleStore[kubernetes.JobSpec]) error {
	podLogger, err := kubernetes.NewPodLogger(&kubeConfig.RestConfig)
	if err != nil {
		return err
	}

	jobWatcher, err := watchers.NewBuildJobWatcher(podLogger, store)
	if err != nil {
		return err
	}
	err = controller.AddWatcher(jobWatcher, kubernetes.JobSchema().Kind())
	if err != nil {
		return err
	}
	jobClient, err := clientGenerator.ClientFor(kubernetes.JobSchema())
	if err != nil {
		return err
	}
	jobInformer, err := operator.NewKubernetesBasedInformer(kubernetes.JobSchema(), jobClient, kubeConfig.Namespace)
	if err != nil {
		return err
	}
	err = controller.AddInformer(jobInformer, kubernetes.JobSchema().Kind())
	if err != nil {
		return err
	}
	return nil
}
