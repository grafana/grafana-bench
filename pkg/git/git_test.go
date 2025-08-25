package git

import (
	"context"
	"path"
	"testing"

	"github.com/grafana/grafana-bench/internal/testutils/gittest"
	"github.com/grafana/grafana-bench/pkg/git/gogit"
)


var repoFiles = map[string][]byte{
	"directory/file":          []byte("file"),
	"anotherDir/another_file": []byte("another file"),
}

func TestGitSource(t *testing.T) {

	testRepo, err := gittest.SetupTestRepo(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("setting up test repo")
	}

	mainBranch := "main"
	_ = testRepo.CreateBranch(mainBranch)
	// if err != nil {
	// 	t.Fatalf("creating branch %s: %v", mainBranch, err)
	// }

	// initialize repo content
	hash, err := testRepo.Commit("add test files", repoFiles)
	if err != nil {
		t.Fatalf("committing files %v", err)
	}

	// create tag
	tagName := "v0.1.0"
	err = testRepo.Tag(tagName, hash)
	if err != nil {
		t.Fatalf("creating tag %v", err)
	}

	// create branch
	branchName := "'test-branch'"
	testRepo.CreateBranch(branchName)

	testCases := []struct {
		name      string
		revision  string
		dirs      []string
		expectErr bool
	}{
		// {
		// 	name:      "get default",
		// 	revision:  "",
		// 	expectErr: false,
		// },
		{
			name:      "get main master",
			revision:  mainBranch,
			expectErr: false,
		},
		// {
		// 	name:      "get branch",
		// 	revision:  branchName,
		// 	expectErr: false,
		// },
		// {
		// 	name:      "get tag",
		// 	revision:  tagName,
		// 	expectErr: false,
		// },
		// {
		// 	name:      "get hash",
		// 	revision:  hash,
		// 	expectErr: false,
		// },
		// {
		// 	name:      "get non-existing hash",
		// 	revision:  "abcdef",
		// 	expectErr: true,
		// },
		// {
		// 	name:      "get non-existing branch",
		// 	revision:  "fake-branch",
		// 	expectErr: true,
		// },
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			source, err := gogit.NewSource(testRepo.URL, testRepo.Token)
			if err != nil {
				t.Fatalf("creating git source %v", err)
			}

			targetDir := path.Join(t.TempDir(), "repo")
			_, err = source.Get(context.TODO(), targetDir, tc.revision)
			if err != nil && !tc.expectErr {
				t.Fatalf("getting source: %v", err)
			}
		})
	}

	// t.Run("get into an existing repository", func(t *testing.T) {
	// 	clonedRepo := filepath.Join(t.TempDir(), "repo")
		
	// 	_, err = git.PlainClone(
	// 		clonedRepo,
	// 		false,
	// 		&git.CloneOptions{
	// 			URL: repoDir,
	// 		},
	// 	)
	// 	if err != nil {
	// 		t.Fatalf("cloning repo %v", err)
	// 	}

	// 	// get source again into cloned repository
	// 	source := NewGitSource(repoDir, "")

	// 	_, err = source.Get(context.TODO(), clonedRepo, "")
	// 	if err == nil {
	// 		t.Fatalf("should have failed")
	// 	}
	// })

	// t.Run("get directories", func(t *testing.T) {
	// 	testCases := []struct {
	// 		title     string
	// 		dirs      []string
	// 		expectErr bool
	// 	}{
	// 		{
	// 			title:     "checkout dir",
	// 			dirs:      []string{"directory"},
	// 			expectErr: false,
	// 		},
	// 		{
	// 			title:     "checkout non-existing dir",
	// 			dirs:      []string{"not-existing-dir"},
	// 			expectErr: true,
	// 		},
	// 	}

	// 	for _, tc := range testCases {
	// 		t.Run(tc.title, func(t *testing.T) {
	// 			// must be in lexicographical order for slices.Equal to work
	// 			dirs := tc.dirs
	// 			slices.Sort(dirs)

	// 			targetRepo := path.Join(t.TempDir(), "repo")
	// 			source := NewGitSource(repoDir, "")

	// 			_, err = source.Get(context.TODO(), targetRepo, "", dirs...)
	// 			if err != nil && !tc.expectErr {
	// 				t.Fatalf("getting source: %v", err)
	// 			}

	// 			if tc.expectErr {
	// 				return
	// 			}

	// 			// collect directories in cloned repo. Exclude .git
	// 			checkedOutDirs := []string{}
	// 			entries, _ := os.ReadDir(targetRepo)
	// 			for _, e := range entries {
	// 				if e.IsDir() && e.Name() != ".git" {
	// 					checkedOutDirs = append(checkedOutDirs, e.Name())
	// 				}
	// 			}

	// 			// check only selected directories were checked out
	// 			if !slices.Equal(dirs, checkedOutDirs) {
	// 				t.Fatalf("expected %v got %v", dirs, checkedOutDirs)
	// 			}
	// 		})
	// 	}
	// })
}
