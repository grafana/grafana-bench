package compile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

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
	TestSuiteRevision string
	RepoToken         string
	TestPrepareCmd    []string
}


func NewTestCompiler(
	log *slog.Logger,
	targetDir string,
	testSuiteRepo string,
	repoToken string,
	testSuiteRevision string,
	testPrepareCmd []string,
)  *TestCompiler {
	return &TestCompiler{
		Log:               log,
		TargetDir:         targetDir,
		TestSuiteRepo:     testSuiteRepo,
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
			NoCheckout: true,
		},
	)

	if err != nil {
		return "", fmt.Errorf("checking out test suite repo %s: %w", tc.TestSuiteRepo, err)
	}

	var tree *git.Worktree

	tree, err = repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting work tree %w", err)
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

	err = tree.Checkout(&git.CheckoutOptions{
		Hash: checkoutHash,
	})
	if err != nil {
		return "", fmt.Errorf("checking out test suite revision %q: %w", tc.TestSuiteRevision, err)
	}

	currentBranch, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("error getting current branch %w", err)
	}

	// set short revision
	revisionHash := currentBranch.Hash().String()[1:7]

	// update repo + checkout branch
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current work directory %w", err)
	}

	// build the tests
	if len(tc.TestPrepareCmd) > 0 {
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
