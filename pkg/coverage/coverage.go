// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"strings"

	"github.com/grafana/grafana-bench/pkg/openapi"
)

var (
	ErrLoadingAPI   = errors.New("loading API")
	ErrPathNotFound = errors.New("path not found")
)

type Path struct {
	Name       string
	Tested     bool
	Operations map[string]bool
	Children   map[string]*Path
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

		// Build tree of elements for the path
		for _, e := range pathElements {
			// matches a subpath?
			child := element.Children[e]
			if child == nil {
				// there's parameter?
				child = element.Children["*"]
			}

			// this path element can be a parameter?
			if child != nil {
				element = child
				continue
			}

			// it is a new sub path or a parameter
			child = &Path{
				Name:       e,
				Children:   map[string]*Path{},
				Operations: map[string]bool{},
			}
			key := e
			// parameters are stored as "*" for easy matching in search
			if strings.Contains(e, "{") {
				key = "*"
			}
			element.Children[key] = child

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

func (p *Path) Find(path string) *Path {
	pathElements := strings.Split(strings.Trim(path, "/"), "/")
	element := p
	for _, e := range pathElements {
		child := element.Children[e]
		if child == nil {
			child = element.Children["*"]
			if child == nil {
				return nil
			}
		}
		element = child
	}
	return element
}

