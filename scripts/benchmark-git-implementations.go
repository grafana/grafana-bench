package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/grafana/grafana-bench/pkg/git"
)

type BenchmarkResult struct {
	Name      string
	Library   string
	Revision  string
	Duration  time.Duration
	Success   bool
	Error     string
}

const (
	testRepoURL         = "https://github.com/grafana/grafana"
	mainCommit          = "a009da2087c473e33d56461f74f4ccd503fd27c7"
	deletedBranchCommit = "b024e87" // This commit should exist in grafana repo
)

func benchmarkGoGit(name, revision string, checkoutDirs ...string) BenchmarkResult {
	result := BenchmarkResult{
		Name:     name,
		Library:  "go-git",
		Revision: revision,
	}

	fmt.Printf("Running %s with go-git...\n", name)

	tmpDir, err := os.MkdirTemp("", "bench-gogit-*")
	if err != nil {
		result.Error = fmt.Sprintf("Error creating temp dir: %v", err)
		return result
	}
	defer os.RemoveAll(tmpDir)

	gitRepo := git.NewGitSource(testRepoURL, "")
	gitRepo.Lg = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})) // Minimal logging
	
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5 minute timeout
	defer cancel()
	
	actualRevision, err := gitRepo.Get(ctx, tmpDir, revision, checkoutDirs...)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Sprintf("Error during checkout: %v", err)
		return result
	}

	result.Success = true
	result.Revision = actualRevision
	fmt.Printf("  ✓ Success: %v, revision: %s\n", result.Duration, actualRevision)
	return result
}

func benchmarkNanoGit(name, revision string, checkoutDirs ...string) BenchmarkResult {
	result := BenchmarkResult{
		Name:     name,
		Library:  "nanogit",
		Revision: revision,
	}

	fmt.Printf("Running %s with nanogit...\n", name)

	tmpDir, err := os.MkdirTemp("", "bench-nanogit-*")
	if err != nil {
		result.Error = fmt.Sprintf("Error creating temp dir: %v", err)
		return result
	}
	defer os.RemoveAll(tmpDir)

	nanoRepo, err := git.NewNanoGitSource(testRepoURL, "")
	if err != nil {
		result.Error = fmt.Sprintf("Error creating nanogit source: %v", err)
		return result
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5 minute timeout
	defer cancel()
	
	actualRevision, err := nanoRepo.Get(ctx, tmpDir, revision, checkoutDirs...)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Sprintf("Error during checkout: %v", err)
		return result
	}

	result.Success = true
	result.Revision = actualRevision
	fmt.Printf("  ✓ Success: %v, revision: %s\n", result.Duration, actualRevision)
	return result
}

func printResults(results []BenchmarkResult) {
	fmt.Printf("\n=== BENCHMARK RESULTS ===\n")
	fmt.Printf("%-40s %-10s %-12s %-15s %s\n", "Test", "Library", "Revision", "Duration", "Status")
	fmt.Printf("%-40s %-10s %-12s %-15s %s\n", "----------------------------------------", "----------", "------------", "---------------", "------")

	for _, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗ " + result.Error
			if len(status) > 30 {
				status = status[:27] + "..."
			}
		}

		fmt.Printf("%-40s %-10s %-12s %-15v %s\n",
			result.Name, result.Library, result.Revision, result.Duration, status)
	}

	// Calculate performance ratios
	fmt.Printf("\n=== PERFORMANCE COMPARISON ===\n")
	for i := 0; i < len(results); i += 2 {
		if i+1 < len(results) && results[i].Success && results[i+1].Success {
			gogit := results[i]
			nanogit := results[i+1]
			
			if gogit.Library != "go-git" {
				gogit, nanogit = nanogit, gogit
			}
			
			ratio := float64(gogit.Duration) / float64(nanogit.Duration)
			fmt.Printf("%-40s: nanogit is %.2fx faster than go-git (%v vs %v)\n",
				gogit.Name, ratio, gogit.Duration, nanogit.Duration)
		}
	}
}

func main() {
	fmt.Printf("Benchmarking go-git vs nanogit implementations\n")
	fmt.Printf("Test repository: %s\n\n", testRepoURL)

	var results []BenchmarkResult

	// Skip full repository tests - too slow for grafana repo
	// Scenario 1: Main branch full checkout
	// fmt.Printf("\n--- Scenario 1: Main branch full checkout ---\n")
	// results = append(results, benchmarkGoGit("Main branch full checkout", ""))
	// results = append(results, benchmarkNanoGit("Main branch full checkout", ""))

	// Scenario 2: Specific commit hash checkout
	// fmt.Printf("\n--- Scenario 2: Specific commit hash checkout ---\n")
	// results = append(results, benchmarkGoGit("Specific commit checkout", mainCommit))
	// results = append(results, benchmarkNanoGit("Specific commit checkout", mainCommit))

	// Scenario 3: Deleted branch commit checkout
	// fmt.Printf("\n--- Scenario 3: Deleted branch commit checkout ---\n")
	// results = append(results, benchmarkGoGit("Deleted branch commit checkout", deletedBranchCommit))
	// results = append(results, benchmarkNanoGit("Deleted branch commit checkout", deletedBranchCommit))

	// Scenario 4: Directory filtering (e2e directory only)
	fmt.Printf("\n--- Scenario 4: Directory filtering (e2e only) ---\n")
	results = append(results, benchmarkGoGit("Directory filtering", "", "e2e"))
	results = append(results, benchmarkNanoGit("Directory filtering", "", "e2e"))

	// Print final results
	printResults(results)
}