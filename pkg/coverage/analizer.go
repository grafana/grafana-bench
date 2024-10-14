package coverage

import (
	"errors"
	"fmt"

	"github.com/grafana/grafana-bench/pkg/openapi"
	"github.com/grafana/grafana-bench/pkg/recorder"
)

var (
	ErrLoadingAPI   = errors.New("loading API")
)

func LoadAPI(rootPath string, prefix string, api openapi.API) (*Path, error) {
	root := NewPath(rootPath)

	for _, path := range api.GetPaths(prefix) {
		pathOps, _ := api.GetOperations(path) // path should exists, don't check err
		ops := []string{}
		for o := range pathOps {
			ops = append(ops, o)
		}
		root.AddSubpath(path, ops...)
	}

	return root, nil
}


type Analizer struct {
	root *Path
}

func NewAnalizer(rootPath string, prefix string, api openapi.API) (*Analizer, error) {
	root, err := LoadAPI(rootPath, prefix, api)
	if err != nil {
		return nil, err
	}

	return &Analizer{root: root}, nil
}

func (a *Analizer) Analize(r recorder.Recording) {
	for _, req := range r.Requests {
		a.root.Record(req.Path, req.Method)
	}
}

func (a *Analizer) Coverage(path string) (Coverage, error) {
	p := a.root.Find(path)
	if p == nil {
		return Coverage{}, fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}

	return p.Coverage(), nil
}