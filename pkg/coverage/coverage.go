// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"strings"
)

var (
	ErrPathNotFound = errors.New("path not found")
)

type EndPoint struct {
	Path       string
	Tested     bool
	Operations map[string]bool
	SubPaths   map[string]*EndPoint
}

type CoverageReport struct {
	Path     string
	Total    int32
	Covered  int32
	Subpaths []CoverageReport
}

func NewEndpoint(path string, operations ...string) *EndPoint {
	ops := map[string]bool{}
	for _, o := range operations {
		ops[o] = false
	}

	return &EndPoint{
		Path:       strings.Trim(path, "/"),
		SubPaths:   map[string]*EndPoint{},
		Operations: ops,
	}
}

func (p *EndPoint) AddSubpath(subpath string, operations ...string) *EndPoint {
	element := p
	pathElements := strings.Split(strings.Trim(subpath, "/"), "/")

	// Build tree of elements for the path
	for _, e := range pathElements {
		// matches a subpath?
		subpath := element.SubPaths[e]
		if subpath == nil {
			// there's parameter?
			subpath = element.SubPaths["*"]
		}

		// this path element can be a parameter?
		if subpath != nil {
			element = subpath
			continue
		}

		// it is a new sub path or a parameter
		subpath = &EndPoint{
			Path:       e,
			SubPaths:   map[string]*EndPoint{},
			Operations: map[string]bool{},
		}
		key := e
		// parameters are stored as "*" for easy matching in search
		if strings.Contains(e, "{") {
			key = "*"
		}
		element.SubPaths[key] = subpath

		element = subpath
	}

	for _, o := range operations {
		element.Operations[o] = false
	}

	// allow chaining multiple call to AddSubpath
	return p
}

func (p *EndPoint) Find(path string) *EndPoint {
	path = strings.Trim(path, "/")

	pathElements := strings.Split(path, "/")
	if pathElements[0] != p.Path {
		return nil
	}

	element := p
	for _, e := range pathElements[1:] {
		subpath := element.SubPaths[e]
		if subpath == nil {
			subpath = element.SubPaths["*"]
			if subpath == nil {
				return nil
			}
		}
		element = subpath
	}
	return element
}

func (p *EndPoint) RecordOperation(path string, op string) {
	endpoint := p.Find(path)
	if endpoint == nil {
		return
	}

	if _, valid := endpoint.Operations[op]; valid {
		endpoint.Operations[op] = true
	}
}

func (p *EndPoint) Coverage() CoverageReport {
	coverage := CoverageReport{
		Path:  p.Path,
		Total: int32(len(p.Operations)),
	}

	// coverage of this path (if it has no operations it does not affect calculation)
	for _, v := range p.Operations {
		if v {
			coverage.Covered = coverage.Covered + 1
		}
	}

	// aggregate coverage of children
	for _, c := range p.SubPaths {
		cc := c.Coverage()
		coverage.Total = coverage.Total + cc.Total
		coverage.Covered = coverage.Covered + cc.Covered
		coverage.Subpaths = append(coverage.Subpaths, cc)
	}

	return coverage

}
