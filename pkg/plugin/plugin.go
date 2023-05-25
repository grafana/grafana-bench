package plugin

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-app-sdk/plugin/kubeconfig"
	"github.com/grafana/grafana-app-sdk/plugin/router"
	"github.com/grafana/grafana-bench/pkg/plugin/secure"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Service interface {
	GetBuildService(context.Context) (BuildService, error)
}

// Plugin is the backend plugin
type Plugin struct {
	router    *router.JSONRouter
	namespace string
	service   Service
}

// Start has the plugin's router start listening over gRPC, and blocks until an unrecoverable error occurs
func (p *Plugin) Start() error {
	return p.router.ListenAndServe()
}

func New(namespace string, service Service) (*Plugin, error) {
	p := &Plugin{
		router:    router.NewJSONRouter(log.DefaultLogger),
		namespace: namespace,
		service:   service,
	}

	p.router.Use(
		kubeconfig.LoadingMiddleware(),
		router.MiddlewareFunc(secure.Middleware))

	// V1 Routes
	v1Subrouter := p.router.Subroute("v1/")

	// Build subrouter
	buildSubrouter := v1Subrouter.Subroute("builds/")
	v1Subrouter.Handle("builds", p.handleBuildList, http.MethodGet)
	v1Subrouter.HandleWithCode("builds", p.handleBuildCreate, http.StatusCreated, http.MethodPost)
	buildSubrouter.Handle("{name}", p.handleBuildGet, http.MethodGet)
	buildSubrouter.Handle("{name}", p.handleBuildUpdate, http.MethodPut)
	buildSubrouter.HandleWithCode("{name}", p.handleBuildDelete, http.StatusNoContent, http.MethodDelete)

	return p, nil
}

type errWithStatusCode interface {
	error
	StatusCode() int
}
