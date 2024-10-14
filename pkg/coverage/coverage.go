// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/grafana/grafana-bench/pkg/openapi"
	"github.com/grafana/grafana-bench/pkg/recorder"
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

type Coverage struct {
	Path     string
	Total    int32
	Covered  int32
	Children []Coverage
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

func (p *Path) Record(path string, op string) {
	element := p.Find(path)
	if element == nil {
		return
	}

	if _, valid := element.Operations[op]; valid {
		element.Operations[op] = true
	}
}

func (p *Path) Coverage() Coverage {
	coverage := Coverage{
		Path:  p.Name,
		Total: int32(len(p.Operations)),
	}

	// coverage of this path (if it has no operations it does not affect calculation)
	for _, v := range p.Operations {
		if v {
			coverage.Covered = coverage.Covered + 1
		}
	}

	// aggregate coverage of children
	for _, c := range p.Children {
		cc := c.Coverage()
		coverage.Total = coverage.Total + cc.Total
		coverage.Covered = coverage.Covered + cc.Covered
		coverage.Children = append(coverage.Children, cc)
	}

	return coverage

}

func (c Coverage) Print(template template.Template, out io.Writer) error {
	err := template.Execute(out, c)
	if err != nil {
		return err
	}
	for _, s := range c.Children {
		err = s.Print(template, out)
		if err != nil {
			return err
		}
	}

	return nil
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
