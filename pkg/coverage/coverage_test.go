package coverage

import (
	"reflect"
	"testing"
)

func TestAddSubpath(t *testing.T) {
	testCases := []struct {
		title    string
		paths    map[string][]string
		expected *EndPoint
	}{
		{
			title: "one path api",
			paths: map[string][]string{
				"/path": []string{"DELETE", "GET"},
			},
			expected: &EndPoint{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndPoint{
					"path": &EndPoint{
						Path: "path",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						SubPaths: map[string]*EndPoint{},
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
			expected: &EndPoint{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndPoint{
					"path1": &EndPoint{
						Path: "path1",
						Operations: map[string]bool{
							"DELETE": false, "POST": false,
						},
						SubPaths: map[string]*EndPoint{},
					},
					"path2": &EndPoint{
						Path: "path2",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						SubPaths: map[string]*EndPoint{},
					},
				},
			},
		},
		{
			title: "path with parameter",
			paths: map[string][]string{
				"/path/{parameter}": []string{"DELETE", "GET"},
			},
			expected: &EndPoint{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndPoint{
					"path": &EndPoint{
						Path:       "path",
						Operations: map[string]bool{},
						SubPaths: map[string]*EndPoint{
							"*": &EndPoint{
								Path: "{parameter}",
								Operations: map[string]bool{
									"DELETE": false, "GET": false,
								},
								SubPaths: map[string]*EndPoint{},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			path := NewEndpoint("api")

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

	// /path/
	// /path/subpath/
	// /path/subpath/{parameter}
	// /path/subpath/{parameter}/subsubpath
	endpoint := NewEndpoint("path").
		AddSubpath("/subpath/{parameter}/subsubpath")

	testCases := []struct {
		title    string
		path     string
		expected *EndPoint
	}{
		{
			title:    "find path",
			path:     "/path",
			expected: endpoint,
		},
		{
			title:    "find non existing path",
			path:     "/another",
			expected: nil,
		},
		{
			title:    "find subpath",
			path:     "/path/subpath",
			expected: endpoint.SubPaths["subpath"],
		},
		{
			title:    "find parameter",
			path:     "/path/subpath/xxxxxxx",
			expected: endpoint.SubPaths["subpath"].SubPaths["*"],
		},
		{
			title:    "find parameter subpath",
			path:     "/path/subpath/xxxxxxx/subsubpath",
			expected: endpoint.SubPaths["subpath"].SubPaths["*"].SubPaths["subsubpath"],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			actual := endpoint.Find(tc.path)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(actual))
			}
		})
	}
}

func TestRecorOperation(t *testing.T) {
	testCases := []struct {
		title string
		// path -> operations
		ops map[string][]string
		// path -> operations (recorded or not)
		expect map[string]map[string]bool
	}{
		{
			title: "one operation at /path/subpath",
			ops: map[string][]string{
				"/path/subpath": []string{"POST"},
			},
			expect: map[string]map[string]bool{
				"/path/subpath": map[string]bool{
					"POST": true,
				},
				"/path/subpath/{parameter}": map[string]bool{
					"GET": false, "DELETE": false,
				},
				"/path/subpath/{parameter}/subsubpath": map[string]bool{
					"GET": false,
				},
			},
		},
		{
			title: "invalid operation",
			ops: map[string][]string{
				"/path/subpath": []string{"GET"},
			},
			expect: map[string]map[string]bool{
				"/path/subpath": map[string]bool{
					"POST": false,
				},
				"/path/subpath/{parameter}": map[string]bool{
					"GET": false, "DELETE": false,
				},
				"/path/subpath/{parameter}/subsubpath": map[string]bool{
					"GET": false,
				},
			},
		},
		{
			title: "different operations on same path",
			ops: map[string][]string{
				"/path/subpath/<parameter>": []string{"GET", "DELETE"},
			},
			expect: map[string]map[string]bool{
				"/path/subpath": map[string]bool{
					"POST": false,
				},
				"/path/subpath/{parameter}": map[string]bool{
					"GET": true, "DELETE": true,
				},
				"/path/subpath/{parameter}/subsubpath": map[string]bool{
					"GET": false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			// /path/
			// /path/subpath/ (POST)
			// /path/subpath/{parameter} (GET, DELETE)
			// /path/subpath/{parameter}/subsubpath (GET)
			path := NewEndpoint("path").
				AddSubpath("/subpath", "POST").
				AddSubpath("/subpath/{parameter}", "GET", "DELETE").
				AddSubpath("/subpath/{parameter}/subsubpath", "GET")

			for p, ops := range tc.ops {
				for _, o := range ops {
					path.RecordOperation(p, o)
				}
			}

			for pathName, expected := range tc.expect {
				target := path.Find(pathName)
				for op, expected := range expected {
					actual := target.Operations[op]
					if actual != expected {
						t.Fatalf("expected %t got %t for %s %s", expected, actual, pathName, op)
					}
				}
			}
		})
	}
}
