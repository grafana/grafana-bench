package coverage

import (
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
func (a *FakeAPI) WithSubPath(path string, operations map[string]string) *FakeAPI {
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

func TestLoadAPI(t *testing.T) {
	testCases := []struct {
		title     string
		api       openapi.API
		prefix    string
		expected  *EndpointTracker
		expectErr error
	}{
		{
			title:  "one path api",
			prefix: "",
			api: NewFakeAPI().
				WithSubPath("/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpointTracker("api").WithSubpath("/path", "DELETE", "GET"),
		},
		{
			title:  "with prefix",
			prefix: "/prefix",
			api: NewFakeAPI().
				WithSubPath("/prefix/path", map[string]string{"DELETE": "Delete", "GET": "get"}).
				WithSubPath("/another/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpointTracker("api").
				WithSubpath("/prefix/path", "DELETE", "GET"),
		},
		{
			title:  "multiple children",
			prefix: "",
			api: NewFakeAPI().
				WithSubPath("/path1", map[string]string{"DELETE": "Delete", "POST": "post"}).
				WithSubPath("/path2", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpointTracker("api").
				WithSubpath("/path1", "DELETE", "POST").
				WithSubpath("/path2", "DELETE", "GET"),
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
