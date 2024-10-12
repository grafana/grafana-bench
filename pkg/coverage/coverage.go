// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"strings"

	"github.com/grafana/grafana-bench/pkg/openapi"
)

var (
	ErrLoadingAPI = errors.New("loading API")
)

type Path struct {
	Name        string
	Tested      bool
	IsParameter bool
	Operations  map[string]bool
	Children    map[string]*Path
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

func LoadAPI(rootPath string, prefix string, api openapi.API) (*Path, error) {
	root := &Path{
		Name:       rootPath,
		Children:   map[string]*Path{},
		Operations: map[string]bool{},
	}

	for _, path := range api.GetPaths(prefix) {
		element := root
		pathElements := strings.Split(strings.Trim(path, "/"), "/")

		// Build tree of elements for the API
		for _, e := range pathElements {
			child := element.Children[e]
			if child != nil {
				element = child
				continue
			}

			isParameter := strings.Contains(e, "{")
			child = &Path{
				Name:        e,
				IsParameter: isParameter,
				Children:    map[string]*Path{},
				Operations:  map[string]bool{},
			}
			element.Children[e] = child
			element = child
		}

		// add operations for the path
		operations, _ := api.GetOperations(path)
		for o := range operations {
			element.Operations[o] = false
		}
	}

	return root, nil
}
