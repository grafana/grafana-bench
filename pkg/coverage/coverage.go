// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"strings"
)

var (
	ErrPathNotFound = errors.New("path not found")
)

// EndpointTracker tracks the operations executed on an endpoint's path or subpath
type EndpointTracker struct {
	Path       string
	Tested     bool
	Operations map[string]bool
	SubPaths   map[string]*EndpointTracker
}

// CoverateReport records the total operations supported and covered by an endpoint and all its subpaths
type CoverageReport struct {
	Path     string
	Total    int32
	Covered  int32
	Subpaths []CoverageReport
}

// NewEndpointTracker creates a new EndpointTracker with a root path
func NewEndpointTracker(path string, operations ...string) *EndpointTracker {
	ops := map[string]bool{}
	for _, o := range operations {
		ops[o] = false
	}

	return &EndpointTracker{
		Path:       strings.Trim(path, "/"),
		SubPaths:   map[string]*EndpointTracker{},
		Operations: ops,
	}
}

// WithSubpath adds a new subpath to the endpoint, creating any parent subpaths as needed.
// Returns a pointer to the Endpoint to allow chaining multiple calls:
// Example, create "root" endpoint with two subpaths
// NewEndpoint("root").
//	       WithSubpath("subpath1", "GET").
//             WithSubpath("subpath2", "GET")
func (p *EndpointTracker) WithSubpath(subpath string, operations ...string) *EndpointTracker {
	p.AddSubpath(subpath, operations...)
	return p
}

// AddSubpath adds a new subpath to the endpoint, creating any parent subpaths as needed.
// Returns a pointer to the newly created endpoint
func (p *EndpointTracker) AddSubpath(subpath string, operations ...string) *EndpointTracker {
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
		subpath = &EndpointTracker{
			Path:       e,
			SubPaths:   map[string]*EndpointTracker{},
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

	return element
}

// Find returns the endpoint that matches the path or nil if not found
func (p *EndpointTracker) Find(path string) *EndpointTracker {
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

// RecordOperation records the execution of an operation in a endpoint
// If the path does does not exists or the operation is not supported, it is ignored
func (p *EndpointTracker) RecordOperation(path string, op string) {
	endpoint := p.Find(path)
	if endpoint == nil {
		return
	}

	if _, valid := endpoint.Operations[op]; valid {
		endpoint.Operations[op] = true
	}
}

// Coverage returns the CoverageReport of the endpoint 
func (p *EndpointTracker) Coverage() CoverageReport {
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
