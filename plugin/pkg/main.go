package main

import (
    "context"
    "fmt"

    "github.com/grafana/grafana-app-sdk/plugin/kubeconfig"
    "github.com/grafana/grafana-app-sdk/resource"
    "github.com/grafana/grafana-app-sdk/k8s"

    "github.com/grafana/grafana-bench/pkg/generated/resource/build"
    "github.com/grafana/grafana-bench/pkg/plugin"
)

func main() {
    svc := &PluginService{}

    // GENERATED SIMPLE SERVICE INITIALIZER CODE
    svc.buildServiceInitializer = kubeconfig.CachingInitializer(
        func(cfg kubeconfig.NamespacedConfig) (plugin.BuildService, error) {
            // This is example code which assumes the API and storage models are identical
            // TODO: REPLACEME
            return resource.NewTypedStore[*build.Object](build.Schema(), k8s.NewClientRegistry(cfg.RestConfig, k8s.ClientConfig{}))
        })
    

    p, err := plugin.New("default", svc) // TODO: fix namespace usage
    if err != nil {
        panic(err)
    }

    // Start listening
    err = p.Start()
    if err != nil {
        panic(err)
    }
}

//
// GENERATED EXAMPLE SERVICE CODE
// You may want to write your own PluginService code. This example code uses lazy-loading initializers
// (as kubeconfig comes from the secureJSONData, which is not known at start-time, only at request-time)
// to return the appropriate service based on the unexported initializer function for each schema Service type.
//

// PluginService implements plugin.Service
type PluginService struct { 
    buildServiceInitializer kubeconfig.Initializer[plugin.BuildService]
}

// GetBuildService returns a BuildService, use the kube config from the context if initialization is required
func (s *PluginService) GetBuildService(ctx context.Context) (plugin.BuildService, error) {
    cfg, err := kubeconfig.FromContext(ctx)
    if err != nil {
        return nil, err
    }

    if s.buildServiceInitializer == nil {
        return nil, fmt.Errorf("no service initialization code")
    }

    return s.buildServiceInitializer(cfg)
}

