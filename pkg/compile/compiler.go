package compile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/grafana/grafana-bench/pkg/utils"
)

// TestCompiler
type TestCompiler struct {
	Log               *slog.Logger
	TargetDir         string
	TestSuiteRepo     string
	CheckoutDirs      []string
	TestSuiteRevision string
	RepoToken         string
	TestPrepareCmd    []string
}


func NewTestCompiler(
	log *slog.Logger,
	targetDir string,
	testSuiteRepo string,
	checkOutDirs []string,
	repoToken string,
	testSuiteRevision string,
	testPrepareCmd []string,
)  *TestCompiler {
	return &TestCompiler{
		Log:               log,
		TargetDir:         targetDir,
		TestSuiteRepo:     testSuiteRepo,
		CheckoutDirs:      checkOutDirs,
		RepoToken:         repoToken,
		TestSuiteRevision: testSuiteRevision,
		TestPrepareCmd:    testPrepareCmd,
	}
}

// CompileTestSuite collect the test suite from a source repository
// returns the test suite revision
func (tc *TestCompiler)CompileTestSuite(ctx context.Context) (string, error) {
	var (
		repo *git.Repository
		err  error
		auth http.AuthMethod
	)

	tc.Log.Debug("cloning test suite")

	if tc.RepoToken != "" {
		// the user is required, but not used. Any non-empty value is accepted (!?)
		auth = &http.BasicAuth{Username: "gituser",Password:  tc.RepoToken}
	}
	repo, err = git.PlainClone(
		tc.TargetDir,
		false,
		&git.CloneOptions{
			URL:      tc.TestSuiteRepo,
			Auth:     auth,
		},
	)

	if err != nil {
		return "", fmt.Errorf("checking out test suite repo %s: %w", tc.TestSuiteRepo, err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("error getting current branch %w", err)
	}

	checkoutHash := head.Hash()

	if tc.TestSuiteRevision != "" {
		// if we are not in the requested branch
		if head.Name().Short() != tc.TestSuiteRevision {
			// fetch remote refs and make them appear as local refs
			// assumes this is a cloned repository with an 'origin' remote
			err = repo.Fetch(&git.FetchOptions{
				RefSpecs: []config.RefSpec{"refs/*:refs/*"},
				Auth: auth,
			})
			if err != nil  && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return "", fmt.Errorf("fetching references %w", err)
			}

			revisionHash, err := repo.ResolveRevision(plumbing.Revision(tc.TestSuiteRevision))
			if err != nil {
				return "", fmt.Errorf("resolving reference to revision %q :%w", tc.TestSuiteRevision, err)
			}
			// ResolveRevision returns &plumbing.Hash
			checkoutHash = *revisionHash
		}
	}

	tree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting work tree %w", err)
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Hash: checkoutHash,
		SparseCheckoutDirectories: tc.CheckoutDirs,
	})
	if err != nil {
		return "", fmt.Errorf("checking out test suite revision %q: %w", tc.TestSuiteRevision, err)
	}

	// if CheckoutDirs was specified, check they are present, git won't report missing dirs
	for _, dir := range tc.CheckoutDirs {
		_, err := os.Stat(filepath.Join(tc.TargetDir, dir))
		if err != nil {
			return "", fmt.Errorf("directory not checked out: %q", dir)
		}
	}

	// set short revision
	revisionHash := checkoutHash.String()[:7]

	// build the tests
	if len(tc.TestPrepareCmd) > 0 {
		workDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current work directory %w", err)
		}

		err = utils.DoInDir(workDir, tc.TargetDir, func() error {
			cmdMake := exec.Command(tc.TestPrepareCmd[0], tc.TestPrepareCmd[1:]...)
			if err := utils.ExecStdout(cmdMake); err != nil {
				return fmt.Errorf("building test suite: %w", err)
			}

			return nil
		})

		return "", err
	}

	return revisionHash, nil
}
