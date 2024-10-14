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

func TestLoadAPI(t *testing.T) {
	testCases := []struct {
		title     string
		api       openapi.API
		prefix    string
		expected  *EndPoint
		expectErr error
	}{
		{
			title:  "one path api",
			prefix: "",
			api: NewFakeAPI().
				WithPath("/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpoint("api").AddSubpath("/path", "DELETE", "GET"),
		},
		{
			title:  "with prefix",
			prefix: "/prefix",
			api: NewFakeAPI().
				WithPath("/prefix/path", map[string]string{"DELETE": "Delete", "GET": "get"}).
				WithPath("/another/path", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpoint("api").
				AddSubpath("/prefix/path", "DELETE", "GET"),
		},
		{
			title:  "multiple children",
			prefix: "",
			api: NewFakeAPI().
				WithPath("/path1", map[string]string{"DELETE": "Delete", "POST": "post"}).
				WithPath("/path2", map[string]string{"DELETE": "Delete", "GET": "get"}),
			expected: NewEndpoint("api").
				AddSubpath("/path1", "DELETE", "POST").
				AddSubpath("/path2", "DELETE", "GET"),
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
