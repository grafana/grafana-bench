package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grafana/grafana-bench/generators/utils"
)

// semVerRegex matches semantic version with optional v prefix. eg: v1.1.1
var semVerRegex = regexp.MustCompile(`v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)

// updateMarkdownDocs updates the bench version to latest
// bench tag across all the docs via pattern
// grafana-bench:vXXXXX
func updateMarkdownDocs(dir string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println("Working dir:", workDir)

	version, err := utils.GetLatestBenchTag(workDir)
	if err != nil {
		return err
	}

	fmt.Println("Latest tag:", version)

	err = updateSemverInMarkdown(dir, version)
	if err != nil {
		return err
	}

	// Also update GitHub Action commit SHA references
	return updateGitHubActionSHA(dir, workDir)
}

// GetLatestTag gets the latest tag from the repo. This is used for getting the latest tag for bench when updated docs

func buildDocVersionRegex() *regexp.Regexp {
	// Define semver pattern
	semverPattern := `v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`
	
	// Create regex that matches either:
	// 1. Prefix patterns: Latest Version: v1.2.3, grafana-bench:v1.2.3, etc.
	// 2. Backtick pattern: `v1.2.3`
	regexStr := `(?:` +
		// Prefix patterns with optional quotes
		`(Latest Version: |grafana-bench:|benchRevision: |bench:|version: )\s*('|\"|\x60)?` + semverPattern + `('|\"|\x60)?` +
		`|` +
		// Standalone backtick pattern
		`\x60` + semverPattern + `\x60` +
		`)`
	
	return regexp.MustCompile(regexStr)
}

// updateSemverInMarkdown walks through the given directory path,
// finds all .md files not prefixed with "bench_",
// and replaces all instances of "grafana-bench:vxxx" with the provided version.
// NOT recursive
func updateSemverInMarkdown(dirPath string, newVersion string) error {

	versionReplacements := []struct {
		Pattern     *regexp.Regexp
		ReplaceFunc func(string) string
	}{

		// Find all semantic versions referenced in the docs
		// Latest Version: v1.1.1
		// grafana-bench:v1.1.1
		// benchRevision: 'v1.1.1'
		// bench:v1.1.1
		// `v1.1.1`
		// version: 'v1.1.1'
		{
			Pattern: buildDocVersionRegex(),
			ReplaceFunc: func(matched string) string {
				// Handle backtick pattern: `v1.2.3`
				if strings.HasPrefix(matched, "`") && strings.HasSuffix(matched, "`") {
					return "`" + newVersion + "`"
				}

				// Handle prefix patterns: Latest Version: v1.2.3, grafana-bench:v1.2.3, etc.
				prefixEnd := strings.Index(matched, ":")
				if prefixEnd == -1 {
					// Fallback: if no colon found, return original
					return matched
				}

				// Include any whitespace after the colon in the prefix
				prefix := matched[:prefixEnd+1]
				for i := prefixEnd + 1; i < len(matched); i++ {
					if matched[i] == ' ' || matched[i] == '\t' {
						prefix += string(matched[i])
					}
				}

				// If we have a quote add that back
				for i := prefixEnd + 1; i < len(matched); i++ {
					if matched[i] == '\'' || matched[i] == '"' || matched[i] == '`' {
						// returns `prefix: "v1.1.1"` with proper surrounding quote
						return prefix + string(matched[i]) + newVersion + string(matched[i])
					}
				}

				// Return the original prefix plus the new version
				return prefix + newVersion
			},
		},
	}

	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// NOTE we're skipping directories.
		// if we decide to do something fancy with paths we'll need to make recursive
		if d.IsDir() {
			return nil
		}

		// file has .md extension and no bench_ prefix
		fileName := filepath.Base(path)
		if !strings.HasSuffix(fileName, ".md") || strings.HasPrefix(fileName, "bench_") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		contentStr := string(content)

		anyReplacements := false
		for _, vr := range versionReplacements {
			if vr.Pattern.MatchString(contentStr) {
				anyReplacements = true
				contentStr = vr.Pattern.ReplaceAllStringFunc(contentStr, vr.ReplaceFunc)
			}
		}

		if anyReplacements {
			err = os.WriteFile(path, []byte(contentStr), 0644)
			if err != nil {
				return fmt.Errorf("failed to write updated content to file %s: %w", path, err)
			}

			fmt.Printf("Updated versions in file: %s\n", path)
		}

		return nil
	})
}

// updateGitHubActionSHA updates GitHub Action commit SHA references in documentation
// to point to the latest commit that modified the action file
func updateGitHubActionSHA(dirPath string, workDir string) error {
	// Get the latest commit SHA that modified the action file
	latestSHA, err := utils.GetLatestCommitForFile(workDir, ".github/actions/setup-grafana-bench/action.yml")
	if err != nil {
		return fmt.Errorf("failed to get latest commit SHA for action file: %w", err)
	}

	fmt.Println("Latest action commit SHA:", latestSHA)

	// Regex to match GitHub Action references with commit SHAs
	// Matches: uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@<40-character-SHA>
	actionSHARegex := regexp.MustCompile(`(uses:\s+grafana/grafana-bench/\.github/actions/setup-grafana-bench@)[a-f0-9]{40}`)

	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process .md files, particularly github_actions.md
		fileName := filepath.Base(path)
		if !strings.HasSuffix(fileName, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		contentStr := string(content)

		if actionSHARegex.MatchString(contentStr) {
			// Replace all occurrences with the latest SHA
			newContentStr := actionSHARegex.ReplaceAllString(contentStr, "${1}"+latestSHA)
			
			err = os.WriteFile(path, []byte(newContentStr), 0644)
			if err != nil {
				return fmt.Errorf("failed to write updated content to file %s: %w", path, err)
			}

			fmt.Printf("Updated GitHub Action SHA in file: %s\n", path)
		}

		return nil
	})
}
