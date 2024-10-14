package coverage

import (
	"reflect"
	"testing"

)

func TestAddSubpath(t *testing.T) {
	testCases := []struct {
		title    string
		paths    map[string][]string
		expected *Path
	}{
		{
			title: "one path api",
			paths: map[string][]string{
				"/path": []string{"DELETE", "GET"},
			},
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
			title: "multiple children",
			paths: map[string][]string{
				"/path1": []string{"DELETE", "POST"},
				"/path2": []string{"DELETE", "GET"},
			},
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
			title: "path with parameter",
			paths: map[string][]string{
				"/path/{parameter}": []string{"DELETE", "GET"},
			},
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
			path := NewPath("api")

			for pathName, operations := range tc.paths {
				path.AddSubpath(pathName, operations...)
			}

			if !reflect.DeepEqual(path, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(path))
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
