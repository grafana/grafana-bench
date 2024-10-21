package coverage

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAddSubpath(t *testing.T) {
	testCases := []struct {
		title    string
		paths    map[string][]string
		expected *EndpointTracker
	}{
		{
			title: "one path api",
			paths: map[string][]string{
				"/path": []string{"DELETE", "GET"},
			},
			expected: &EndpointTracker{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndpointTracker{
					"path": &EndpointTracker{
						Path: "path",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						SubPaths: map[string]*EndpointTracker{},
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
			expected: &EndpointTracker{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndpointTracker{
					"path1": &EndpointTracker{
						Path: "path1",
						Operations: map[string]bool{
							"DELETE": false, "POST": false,
						},
						SubPaths: map[string]*EndpointTracker{},
					},
					"path2": &EndpointTracker{
						Path: "path2",
						Operations: map[string]bool{
							"DELETE": false, "GET": false,
						},
						SubPaths: map[string]*EndpointTracker{},
					},
				},
			},
		},
		{
			title: "path with parameter",
			paths: map[string][]string{
				"/path/{parameter}": []string{"DELETE", "GET"},
			},
			expected: &EndpointTracker{
				Path:       "api",
				Operations: map[string]bool{},
				SubPaths: map[string]*EndpointTracker{
					"path": &EndpointTracker{
						Path:       "path",
						Operations: map[string]bool{},
						SubPaths: map[string]*EndpointTracker{
							"*": &EndpointTracker{
								Path: "{parameter}",
								Operations: map[string]bool{
									"DELETE": false, "GET": false,
								},
								SubPaths: map[string]*EndpointTracker{},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			path := NewEndpointTracker("api")

			for pathName, operations := range tc.paths {
				path.WithSubpath(pathName, operations...)
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
	endpoint := NewEndpointTracker("/path").
		WithSubpath("/subpath/{parameter}/subsubpath")

	testCases := []struct {
		title    string
		path     string
		expected *EndpointTracker
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
			endpoint := NewEndpointTracker("path").
				WithSubpath("/subpath", "POST").
				WithSubpath("/subpath/{parameter}", "GET", "DELETE").
				WithSubpath("/subpath/{parameter}/subsubpath", "GET")

			for p, ops := range tc.ops {
				for _, o := range ops {
					endpoint.RecordOperation(p, o)
				}
			}

			for pathName, expected := range tc.expect {
				target := endpoint.Find(pathName)
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

func TestCoverage(t *testing.T) {
	testCases := []struct {
		title string
		// path -> operations
		ops      map[string][]string
		expected CoverageReport
	}{
		{
			title: "one operation at /path",
			ops: map[string][]string{
				"/path": []string{"POST"},
			},
			expected: CoverageReport{
				Path:    "path",
				Total:   4,
				Covered: 1,
				Coverage: 25,
				Subpaths: []CoverageReport{
					{
						Path:    "{parameter}",
						Total:   3,
						Covered: 0,
						Coverage: 0,
						Subpaths: []CoverageReport{
							{
								Path:    "subpath",
								Total:   1,
								Covered: 0,
								Coverage: 0,
							},
						},
					},
				},
			},
		},
		{
			title: "one operation at /path/{parameter/subpath}",
			ops: map[string][]string{
				"/path/<value>/subpath": []string{"GET"},
			},
			expected: CoverageReport{
				Path:    "path",
				Total:   4,
				Covered: 1,
				Coverage: 25,
				Subpaths: []CoverageReport{
					{
						Path:    "{parameter}",
						Total:   3,
						Covered: 1,
						Coverage: 33,
						Subpaths: []CoverageReport{
							{
								Path:    "subpath",
								Total:   1,
								Covered: 1,
								Coverage: 100,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			// /path  (POST)
			// /path/{parameter} (GET, DELETE)
			// /path/{parameter}/subpath (GET)
			endpoint := NewEndpointTracker("path", "POST").
				WithSubpath("/{parameter}", "GET", "DELETE").
				WithSubpath("/{parameter}/subpath", "GET")

			for p, ops := range tc.ops {
				for _, o := range ops {
					endpoint.RecordOperation(p, o)
				}
			}

			actual := endpoint.Coverage()
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(actual))
			}
		})
	}
}


func TestPrint(t *testing.T) {
	testCases := []struct {
		title string
		opts PrintOptions
		expected string
	}{
		{
			title: "print depth 1",
			opts: PrintOptions {
				MaxDepth: 1,
			},
			expected: "/path 25% (1/4)\n",
		},
		{
			title: "print depth 2",
			opts: PrintOptions {
				MaxDepth: 2,
			},
			expected: "/path 25% (1/4)\n" +
				"/path/subpath1 0% (0/3)\n" +
				"/path/subpath2 0% (0/2)\n",

		},
		{
			title: "print indented",
			opts: PrintOptions {
				MaxDepth: 2,
				Indent: true,
			},
			expected: "/path 25% (1/4)\n" +
				"    /subpath1 0% (0/3)\n" +
				"    /subpath2 0% (0/2)\n",

		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			report := CoverageReport{
				Path:    "path",
				Total:   4,
				Covered: 1,
				Coverage: 25,
				Subpaths: []CoverageReport{
					{
						Path:    "subpath1",
						Total:   3,
						Covered: 0,
						Coverage: 0,
						Subpaths: []CoverageReport{
							{
								Path:    "subsubpath",
								Total:   1,
								Covered: 0,
								Coverage: 0,
							},
						},
					},
					{
						Path:    "subpath2",
						Total:   2,
						Covered: 0,
						Coverage: 0,
						Subpaths: []CoverageReport{},
					},
				},
			}

			buffer := &bytes.Buffer{}
			report.Print(tc.opts, buffer)
			actual := buffer.String()
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("expected %s got %s", deepPrint(tc.expected), deepPrint(actual))
			}
		})
	}
}
