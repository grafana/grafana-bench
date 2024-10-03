package compile

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const makeFile = `
build:
	echo "building"
fail:
	echo "failed"
	/bin/true
`

func Test_Compiler(t *testing.T) {

	repoDir := t.TempDir()
	r, err := git.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("initializing repository: %v", err)
	}

	// crate Makefile in repository
	err = os.WriteFile(path.Join(repoDir, "Makefile"), []byte(makeFile), 0o644)
	if err != nil {
		t.Fatalf("copying file to repository: %v", err)
	}

	// 3. commit files (path must be relative to repo's root)
	wt, _ := r.Worktree()
	_, err = wt.Add(".")
	if err != nil {
		t.Fatalf("adding files to commit: %v", err)
	}

	_, err = wt.Commit("add test files", &git.CommitOptions{
		Author: &object.Signature{Name: "grafana bench", Email: "bench-testing@grafana.com"},
	})
	if err != nil {
		t.Fatalf("committing files: %v", err)
	}

	testCases := []struct {
		name       string
		revision   string
		prepareCmd []string
		expectErr  bool
	}{
		{
			name:       "execute prepare command",
			revision:   "master",
			expectErr:  false,
			prepareCmd: []string{"make", "build"},
		},
		{
			name:       "execute failing prepare command",
			revision:   "master",
			expectErr:  true,
			prepareCmd: []string{"make", "fail"},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			logBuffer := bytes.Buffer{}
			log := slog.New(slog.NewTextHandler(&logBuffer, nil))

			compiler := NewTestCompiler(
				log,
				path.Join(t.TempDir(), "repo"),
				repoDir,
				[]string{},
				"",
				tc.revision,
				tc.prepareCmd,
			)

			_, err := compiler.CompileTestSuite(context.TODO())
			if err != nil && !tc.expectErr {
				t.Fatalf("compiling test: %v", err)
			}
		})
	}

}
