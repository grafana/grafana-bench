package coverage

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/grafana/grafana-bench/pkg/openapi"
)

type FakeAPI struct {
	paths map[string]map[string]string
}

func NewFakeAPI() *FakeAPI {
	return &FakeAPI{
		paths: map[string]map[string]string{},
	}
}
func (a *FakeAPI) WithPath(path string, operations map[string]string) *FakeAPI {
	a.paths[path] = operations
	return a
}

func (a *FakeAPI) GetPaths(prefix string) []string {
	paths := []string{}
	for p := range a.paths {
		if strings.HasPrefix(p, prefix) {
			paths = append(paths, p)
		}
	}

	slices.Sort(paths)

	return paths
}

func (a *FakeAPI) GetOperations(path string) (map[string]string, error) {
	operations := a.paths[path]
	if operations == nil {
		return nil, openapi.ErrPathNotFound
	}

	return operations, nil
}

func deepPrint(value any) string {
	s, _ := json.MarshalIndent(value, "", "\t")
	return string(s)
}

func TestLoadAPI(t *testing.T) {
	testCases := []struct {
		title     string
		api       openapi.API
		prefix    string
		expected  *Path
		expectErr error
	}{
		{
			title:  "one path api",
			prefix: "",
			api: NewFakeAPI().
				WithPath("/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: &Path{
				Name:       "api",
				Operations: map[string]bool{},
				Children: map[string]*Path{
					"path": &Path{
						Name: "path",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						Children: map[string]*Path{},
					},
				},
			},
		},
		{
			title:  "with prefix",
			prefix: "/prefix",
			api: NewFakeAPI().
				WithPath("/prefix/path", map[string]string{"DELETE": "Delete", "GET": "get"}).
				WithPath("/another/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: &Path{
				Name:       "api",
				Operations: map[string]bool{},
				Children: map[string]*Path{
					"prefix": &Path{
						Name:       "prefix",
						Operations: map[string]bool{},
						Children: map[string]*Path{
							"path": &Path{
								Name: "path",
								Operations: map[string]bool{
									"DELETE": false, "GET": false,
								},
								Children: map[string]*Path{},
							},
						},
					},
				},
			},
		},
		{
			title:  "multiple children",
			prefix: "",
			api: NewFakeAPI().
				WithPath("/path1", map[string]string{"DELETE": "Delete", "POST": "post"}).
				WithPath("/path2", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: &Path{
				Name:       "api",
				Operations: map[string]bool{},
				Children: map[string]*Path{
					"path1": &Path{
						Name: "path1",
						Operations: map[string]bool{
							"DELETE": false, "POST": false,
						},
						Children: map[string]*Path{},
					},
					"path2": &Path{
						Name: "path2",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						Children: map[string]*Path{},
					},
				},
			},
		},
		{
			title:  "path with parameter",
			prefix: "",
			api: NewFakeAPI().
				WithPath("/path/{parameter}", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: &Path{
				Name:       "api",
				Operations: map[string]bool{},
				Children: map[string]*Path{
					"path": &Path{
						Name:       "path",
						Operations: map[string]bool{},
						Children: map[string]*Path{
							"*": &Path{
								Name: "{parameter}",
								Operations: map[string]bool{
									"DELETE": false, "GET": false,
								},
								Children: map[string]*Path{},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			actual, err := LoadAPI("api", tc.prefix, tc.api)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected error %v got %v", tc.expectErr, err)
			}

			if tc.expectErr != nil {
				return
			}

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(actual))
			}
		})
	}
}

func TestFind(t *testing.T) {

	// /path/subpath
	// /path/{parameter}
	// /path/{parameter}/subsubpath
	path := &Path{
		Name:       "api",
		Operations: map[string]bool{},
		Children: map[string]*Path{
			"path": &Path{
				Name:       "path",
				Operations: map[string]bool{},
				Children: map[string]*Path{
					"subpath": &Path{
						Name:     "subpath",
						Children: map[string]*Path{},
					},
					"*": &Path{
						Name: "{parameter}",
						Children: map[string]*Path{
							"subsubpath": &Path{
								Name:     "subsubpath",
								Children: map[string]*Path{},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		title    string
		path     string
		expected *Path
	}{
		{
			title:    "find path",
			path:     "/path",
			expected: path.Children["path"],
		},
		{
			title:    "find non existing path",
			path:     "/another",
			expected: nil,
		},
		{
			title:    "find subpath",
			path:     "/path/subpath",
			expected: path.Children["path"].Children["subpath"],
		},
		{
			title:    "find parameter",
			path:     "/path/xxxxxxx",
			expected: path.Children["path"].Children["*"],
		},
		{
			title:    "find parameter subpath",
			path:     "/path/xxxxxxx/subsubpath",
			expected: path.Children["path"].Children["*"].Children["subsubpath"],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			actual := path.Find(tc.path)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(actual))
			}
		})
	}
}
