// Package coverage implements function for calculating test coverage
package coverage

import (
	"errors"
	"fmt"
	"io"
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
	Coverage int32
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
		element.Operations[strings.ToUpper(o)] = false
	}

	return element
}

// Find returns the endpoint that matches the path or nil if not found
func (p *EndpointTracker) Find(path string) *EndpointTracker {
	path = strings.Trim(path, "/")

	// path must start's with the endpoint's path
	path, found := strings.CutPrefix(path, p.Path)
	if !found {
		return nil
	}

	pathElements := strings.Split(path, "/")

	element := p
	for _, e := range pathElements {
		// skip any empty element that Split returned
		if e == "" {
			continue
		}
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

	op = strings.ToUpper(op)
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

	if coverage.Total > 0 {
		coverage.Coverage = int32(100.0 * float32(coverage.Covered)/float32(coverage.Total))
	}

	return coverage
}


type VisitState struct {
	Depth    int
	FullPath []string
	Any      any
}

// Visit executes a visitor function recursively on a Coverage Report for a path and its subpaths
// until it returns false. state argument can be used to carry information along the process (e.g. calculate total)
func (c CoverageReport) Visit( visitor func(c CoverageReport, state VisitState) bool, any any) {
	state := VisitState{
		Depth: 1,
		FullPath: []string{c.Path},
		Any: any,
	}

	c.doVisit(visitor, state)
}

func (c CoverageReport) doVisit( visitor func(c CoverageReport, s VisitState) bool, state VisitState) bool {
	if visitor(c, state) {
		for _, p := range c.Subpaths {
			vs := VisitState{
				Depth: state.Depth+1,
				FullPath: append(state.FullPath, p.Path), 
				Any: state.Any,
			}
			p.doVisit(visitor, vs)
		}
		return true
	}

	return false
}

type PrintOptions struct {
	MaxDepth      int
	Indent        bool
	SkipUncovered bool
}
func (c  CoverageReport)Print(opts PrintOptions, writer io.Writer) {
	c.Visit(func(c CoverageReport, state VisitState) bool {
		if (opts.MaxDepth > 0 && state.Depth > opts.MaxDepth) || (c.Covered == 0 && opts.SkipUncovered) {
			return false
		}

		path := ""
		if opts.Indent {
			parentPath := strings.Join(state.FullPath[:state.Depth-1], "/")
			path = fmt.Sprintf("%s/%s", strings.Repeat(" ", len(parentPath)), c.Path)
		} else {
			path = "/"+strings.Join(state.FullPath, "/")
		}
		fmt.Fprintf(writer, "%s %d%% (%d/%d)\n", path, c.Coverage, c.Covered, c.Total)
		
		return true
	}, nil)
}