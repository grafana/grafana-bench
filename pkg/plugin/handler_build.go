package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/grafana-app-sdk/plugin"
	"github.com/grafana/grafana-app-sdk/plugin/router"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana-bench/pkg/generated/resource/build"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type BuildService interface {
	List(ctx context.Context, namespace string, filters ...string) (*resource.TypedStoreList[*build.Object], error)
	Get(ctx context.Context, id resource.Identifier) (*build.Object, error)
	Add(ctx context.Context, obj *build.Object) (*build.Object, error)
	Update(ctx context.Context, id resource.Identifier, obj *build.Object) (*build.Object, error)
	Delete(ctx context.Context, id resource.Identifier) error
}

func (p *Plugin) handleBuildList(ctx context.Context, req router.JSONRequest) (router.JSONResponse, error) {
	filtersRaw := req.URL.Query().Get("filters")
	filters := make([]string, 0)
	if len(filtersRaw) > 0 {
		filters = strings.Split(filtersRaw, ",")
	}
	svc, err := p.service.GetBuildService(ctx)
	if err != nil {
		log.DefaultLogger.Error("Error getting BuildService: " + err.Error())
		return nil, plugin.NewError(http.StatusInternalServerError, err.Error())
	}
	return svc.List(ctx, p.namespace, filters...)
}

func (p *Plugin) handleBuildGet(ctx context.Context, req router.JSONRequest) (router.JSONResponse, error) {
	svc, err := p.service.GetBuildService(ctx)
	if err != nil {
		log.DefaultLogger.Error("Error getting BuildService: " + err.Error())
		return nil, plugin.NewError(http.StatusInternalServerError, err.Error())
	}
	obj, err := svc.Get(ctx, resource.Identifier{
		Namespace: p.namespace,
		Name:      req.Vars.MustGet("name"),
	})
	if err != nil {
		if e, ok := err.(errWithStatusCode); ok {
			return nil, plugin.NewError(e.StatusCode(), e.Error())
		} else {
			log.DefaultLogger.Error("Error getting Build '" + req.Vars.MustGet("name") + "': " + err.Error())
		}
	}
	return obj, err
}

func (p *Plugin) handleBuildCreate(ctx context.Context, req router.JSONRequest) (router.JSONResponse, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, plugin.NewError(http.StatusBadRequest, err.Error())
	}

	t := build.Object{}
	// TODO: this should eventually be unmarshalled via a method in the Object itself, so Thema can handle it
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, plugin.NewError(http.StatusBadRequest, err.Error())
	}

	svc, err := p.service.GetBuildService(ctx)
	if err != nil {
		log.DefaultLogger.Error("Error getting BuildService: " + err.Error())
		return nil, plugin.NewError(http.StatusInternalServerError, err.Error())
	}
	t.StaticMeta.Namespace = p.namespace
	obj, err := svc.Add(ctx, &t)
	if err != nil {
		if e, ok := err.(errWithStatusCode); ok {
			return nil, plugin.NewError(e.StatusCode(), e.Error())
		} else {
			log.DefaultLogger.Error("Error creating new Build: " + err.Error())
		}
	}
	return obj, err
}

func (p *Plugin) handleBuildUpdate(ctx context.Context, req router.JSONRequest) (router.JSONResponse, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, plugin.NewError(http.StatusBadRequest, err.Error())
	}

	t := build.Object{}
	// TODO: this should eventually be unmarshalled via a method in the Object itself, so Thema can handle it
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, plugin.NewError(http.StatusBadRequest, err.Error())
	}

	svc, err := p.service.GetBuildService(ctx)
	if err != nil {
		log.DefaultLogger.Error("Error getting BuildService: " + err.Error())
		return nil, plugin.NewError(http.StatusInternalServerError, err.Error())
	}
	obj, err := svc.Update(ctx, resource.Identifier{
		Namespace: p.namespace,
		Name:      req.Vars.MustGet("name"),
	}, &t)
	if err != nil {
		if e, ok := err.(errWithStatusCode); ok {
			return nil, plugin.NewError(e.StatusCode(), e.Error())
		} else {
			log.DefaultLogger.Error("Error updating Build '" + req.Vars.MustGet("name") + "': " + err.Error())
		}
	}
	return obj, err
}

func (p *Plugin) handleBuildDelete(ctx context.Context, req router.JSONRequest) (router.JSONResponse, error) {
	svc, err := p.service.GetBuildService(ctx)
	if err != nil {
		log.DefaultLogger.Error("Error getting BuildService: " + err.Error())
		return nil, plugin.NewError(http.StatusInternalServerError, err.Error())
	}
	err = svc.Delete(ctx, resource.Identifier{
		Namespace: p.namespace,
		Name:      req.Vars.MustGet("name"),
	})
	if err != nil {
		if e, ok := err.(errWithStatusCode); ok {
			return nil, plugin.NewError(e.StatusCode(), e.Error())
		} else {
			log.DefaultLogger.Error("Error deleting Build '" + req.Vars.MustGet("name") + "': " + err.Error())
		}
	}
	return nil, err
}
