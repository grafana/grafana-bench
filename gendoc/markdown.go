package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

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

	version, err := getLatestBenchTag(workDir)
	if err != nil {
		return err
	}

	fmt.Println("Latest tag:", version)

	return updateSemverInMarkdown(dir, version)
}

// GetLatestTag gets the latest tag from the repo. This is used for getting the latest tag for bench when updated docs
func getLatestBenchTag(repoPath string) (string, error) {
	// Open the repository
	repo, err := git.PlainOpenWithOptions(
		repoPath,
		&git.PlainOpenOptions{DetectDotGit: true},
	)
	if err != nil {
		return "", err
	}

	// Get all tags
	tagsIter, err := repo.Tags()
	if err != nil {
		return "", err
	}

	// Create a slice to store tag information
	type tagInfo struct {
		Name string
		Time time.Time
	}
	var tags []tagInfo

	// Collect all tags with their commit time
	err = tagsIter.ForEach(func(ref *plumbing.Reference) error {
		// Get the tag object
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			// If it's an annotated tag, use its timestamp
			tags = append(tags, tagInfo{
				Name: ref.Name().Short(),
				Time: tagObj.Tagger.When,
			})
			return nil
		}

		// For lightweight tags, get the commit it points to
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			// If we can't get the commit, just use the current time as a fallback
			tags = append(tags, tagInfo{
				Name: ref.Name().Short(),
				Time: time.Now(),
			})
			return nil
		}

		tags = append(tags, tagInfo{
			Name: ref.Name().Short(),
			Time: commit.Committer.When,
		})
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repository")
	}

	// Sort tags by time, newest first
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Time.After(tags[j].Time)
	})

	// get the latest tag that matches semver
	for _, t := range tags {
		if semVerRegex.MatchString(t.Name) {
			return t.Name, nil
		}
	}

	// Return the latest tag name
	return "", fmt.Errorf("no tags that meet semantic versioning standards found in repo")
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

		// find all semantic versions referenced in the docs
		{
			Pattern: regexp.MustCompile(`(Latest Version:|grafana-bench:|benchRrevision:|bench:)\s*v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`),
			ReplaceFunc: func(matched string) string {
				// Find the index where the prefix ends (after the colon and whitespace)
				prefixEnd := strings.Index(matched, ":")
				if prefixEnd == -1 {
					return matched // Should never happen with our regex
				}

				// Include any whitespace after the colon in the prefix
				prefix := matched[:prefixEnd+1]
				for i := prefixEnd + 1; i < len(matched); i++ {
					if matched[i] == ' ' || matched[i] == '\t' {
						prefix += string(matched[i])
					}
				}

				// Return the original prefix plus the new version
				return prefix + newVersion
			},
		},

		// index reference
		//{
		//  Pattern:     regexp.MustCompile(`Latest Version: v[0-9.]+`),
		//  Replacement: "Latest Version: " + newVersion,
		//},
		// bench image reference
		//{
		//  Pattern:     regexp.MustCompile(`grafana-bench:v[^\s\n\r\t,'"]*`),
		//  Replacement: "grafana-bench:" + newVersion,
		//},
		// libsonnet version reference
		//{
		//  Pattern:     regexp.MustCompile(`benchRevision: ['"]v[^'"]*['"],`),
		//  Replacement: "benchRevision: '" + newVersion + "',",
		//},
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
