// Package openapi implements functions to inspect OpenAPI specifications
package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrInvalidDocument = errors.New("invalid document")
	ErrInvalidSource   = errors.New("invalid source")
	ErrPathNotFound    = errors.New("path not found")
	
)
// V3Operation defines a operation in the OpenAPI 3.0 specification
type V3Operation struct {
	Description string `json:"description,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// V3Path defines a path in the OpenAPI 3.0 specification
type V3Path struct {
	Description string       `json:"description,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Delete      *V3Operation `json:"delete,omitempty"`
	Get         *V3Operation `json:"get,omitempty"`
	Put         *V3Operation `json:"put,omitempty"`
	Post        *V3Operation `json:"post,omitempty"`
	Patch       *V3Operation `json:"patch,omitempty"`
}

func (a *V3API) GetPaths(prefix string) []string {
	var paths []string
	for p := range a.Paths {
		if strings.HasPrefix(p, prefix) {
			paths = append(paths, p)
		}
	}
	return paths
}

// GetOperations returns a map of operations in a V3Path
func (a *V3API) GetOperations(path string) (map[string]string, error)  {
	p := a.Paths[path]
	if p == (V3Path{}) {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}
	operations := make(map[string]string)

	if p.Delete != nil {
		operations["delete"] = p.Delete.Summary
	}
	if p.Get != nil {
		operations["get"] = p.Get.Summary
	}
	if p.Put != nil {
		operations["put"] = p.Put.Summary
	}
	if p.Post != nil {
		operations["post"] = p.Post.Summary
	}
	if p.Patch != nil {
		operations["patch"] = p.Patch.Summary
	}

	return operations, nil
}

// V3API defines a document in the OpenAPI 3.0 specification
type V3API struct {
	Version string            `json:"version"`
	Paths   map[string]V3Path `json:"paths"`
}


// API defines an interface for extracting information from a OpenAPI specification
type API interface {
	// GetPaths returns a sorted list of paths in the API that matches the prefix 
	GetPaths(prefix string) []string
	// GetOperations returns a list of operations in the API that matches the path with their description
	GetOperations(path string) (map[string]string, error)
}

func FromFile(path string) (API, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read file: %w", ErrInvalidSource, err)
	}

	doc := V3API{}
	err = json.Unmarshal(content, &doc)
	if err != nil {
		return nil, fmt.Errorf("%w: parsing specs: %w", ErrInvalidDocument, err)
	}

	return &doc, nil
}
