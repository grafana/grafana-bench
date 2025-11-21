package compile

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/grafana/grafana-bench/internal/testutils/gittest"
)

const makeFile = `
build:
	echo "building"
fail:
	echo "failed"
	/bin/false
`

func Test_Compiler(t *testing.T) {
	testRepo, err := gittest.SetupTestRepo(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("setting up test repo")
	}

	// initialize repo content
	files := map[string][]byte{
		"Makefile": []byte(makeFile),
	}
	_, err = testRepo.Commit("add test files", files)
	if err != nil {
		t.Fatalf("committing files %v", err)
	}

	// push changes
	err = testRepo.Push()
	if err != nil {
		t.Fatalf("pushing changes %v", err)
	}

	testCases := []struct {
		name       string
		prepareCmd []string
		expectErr  bool
	}{
		{
			name:       "execute prepare command",
			expectErr:  false,
			prepareCmd: []string{"make", "build"},
		},
		{
			name:       "execute failing prepare command",
			expectErr:  true,
			prepareCmd: []string{"make", "fail"},
		},
	}

	drivers := []string{"nonogit", "gogit"}
	for _, driver := range drivers {
		for _, tc := range testCases {
			tc := tc
			name := fmt.Sprintf("%s with %s", tc.name, driver)
			t.Run(name, func(t *testing.T) {
				logBuffer := bytes.Buffer{}
				log := slog.New(slog.NewTextHandler(&logBuffer, nil))

				compiler := NewTestCompiler(
					log,
					driver,
					t.TempDir(),
					testRepo.URL,
					[]string{},
					testRepo.Token,
					"master",
					tc.prepareCmd,
				)

				_, err := compiler.CompileTestSuite(context.TODO())
				if err != nil && !tc.expectErr {
					t.Fatalf("compiling test: %v", err)
				}
			})
		}
	}

}
