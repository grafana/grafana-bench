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
	"github.com/grafana/grafana-bench/pkg/utils"
)

var TargetCloneDir = "./test-repo"

// TestCompiler
type TestCompiler struct {
	Log               *slog.Logger
	TargetDir         string
	TestSuiteRepo     string
	TestSuiteRevision string
}

func NewTestCompiler(
	log *slog.Logger,
	targetDir string,
	testSuiteRepo string,
	testSuiteRevision string,
) *TestCompiler {
	return &TestCompiler{
		Log:               log,
		TargetDir:         targetDir,
		TestSuiteRepo:     testSuiteRepo,
		TestSuiteRevision: testSuiteRevision,
	}
}

func (tc *TestCompiler) CloneRepo(ctx context.Context) error {
	var (
		repo *git.Repository
		err  error
	)

	// clone repo if doesn't exist
	exists, _ := utils.PathExists(tc.TargetDir)
	if exists {
		repo, err = git.PlainOpen(tc.TargetDir)
		if err != nil {
			return fmt.Errorf("opening repo %s: %w", tc.TestSuiteRepo, err)
		}

	} else {
		tc.Log.Info("cloning test suite")
		repo, err = git.PlainClone(
			tc.TargetDir,
			false,
			&git.CloneOptions{
				URL:      tc.TestSuiteRepo,
				Progress: os.Stdout,
			},
		)

		if err != nil {
			return fmt.Errorf("checking out test suite repo %s: %w", tc.TestSuiteRepo, err)
		}
	}

	// if we don't specify a revision, assume we want to run exactly what is
	// there. e.g. local development. Otherwise, proceed to checkout the revision
	if tc.TestSuiteRevision != "" {
		// check current branch. Don't do anything if it's the same as what is
		// currently there.
		// TODO: this logic may not make sense in scenarios where it's set to main
		// but someone hasn't checked out in a while and wants to update. That would
		// require a manual update. perhaps check the git sha. review later.
		var branch *plumbing.Reference

		branch, err = repo.Head()
		if err != nil {
			return fmt.Errorf("error getting current branch %w", err)
		}

		// if we are not in the requested branch
		if branch.Name().Short() != tc.TestSuiteRevision {
			// fetch remote refs and make them appear as local refs
			// assumes this is a cloned repository with an 'origin' remote
			err = repo.Fetch(&git.FetchOptions{
				RefSpecs: []config.RefSpec{"refs/*:refs/*"},
			})
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return fmt.Errorf("fetching references %w", err)
			}

			var tree *git.Worktree

			tree, err = repo.Worktree()
			if err != nil {
				return fmt.Errorf("getting work tree %w", err)
			}

			repo.References()

			revisionHash, err := repo.ResolveRevision(plumbing.Revision(tc.TestSuiteRevision))
			if err != nil {
				return fmt.Errorf("resolving reference to revision %q :%w", tc.TestSuiteRevision, err)
			}

			// FIXME: this only works for remote branches. Local branches are not found due to the reference
			err = tree.Checkout(&git.CheckoutOptions{
				Hash: *revisionHash,
			})
			if err != nil {
				return fmt.Errorf("checking out test suite revision %q: %w", tc.TestSuiteRevision, err)
			}
		}
	}

	return nil
}

// CompileTestSuite collects and builds tests from a source repository
func (tc *TestCompiler) CompileTestSuite(ctx context.Context) error {
	tc.CloneRepo(ctx)

	// update repo + checkout branch
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current work directory %w", err)
	}

	// build the tests
	err = utils.DoInDir(workDir, tc.TargetDir, func() error {
		cmdMake := exec.Command("make", "build")
		if err := utils.ExecStdout(cmdMake); err != nil {
			return fmt.Errorf("building test suite: %w", err)
		}

		return nil
	})

	return err
}
