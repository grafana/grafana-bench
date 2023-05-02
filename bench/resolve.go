package bench

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
)

func (b *Config) ResolveGrafanaCommit() error {
	if b.GrafanaRevision == "" {
		b.GrafanaRevision = "main"
	}

	// if already a commit hash, we can just return
	if utils.IsCommitHash(b.GrafanaRevision) {
		fmt.Println("grafana: revision", b.GrafanaRevision)
		return nil
	}

	// get latest commit from branch
	branch := b.GrafanaRevision
	fmt.Println("grafana: branch", branch, "specified. Resolving latest commit")
	commit, err := utils.GetLatestBranchCommit("https://github.com/grafana/grafana", b.GrafanaRevision)
	if err != nil {
		return err
	}

	b.GrafanaRevision = commit
	fmt.Printf("grafana: branch %s resolved to `%s`\n", branch, b.GrafanaRevision)
	return nil
}

func (b *Config) ResolveArch() error {
	// Get GoEnv from system
	b.GoEnv = utils.GetCompilerEnvInfo()

	// TODO maybe handle the case where key is not found. probably just a panic
	if b.Arch == "" {
		sys_os := b.GoEnv["GOOS"]
		sys_arch := b.GoEnv["GOARCH"]
		b.Arch = fmt.Sprintf("%s/%s", strings.ToLower(sys_os), strings.ToLower(sys_arch))
	}

	fmt.Println("arch:", b.Arch)

	return nil
}

func (b *Config) ResolveGrafanaINI() error {
	// Check if INI environment set and use that
	// If no INI check to see if local custom.ini in root
	// else blank

	// check if INI is set
	if b.GrafanaINIPath != "" {
		if !filepath.IsAbs(b.GrafanaINIPath) {
			b.GrafanaINIPath = path.Join(b.ProjectRoot, b.GrafanaINIPath)
		}

		exists, _ := utils.PathExists(b.GrafanaINIPath)
		if !exists {
			return fmt.Errorf("grafana-config: error specified config file not found %s", b.GrafanaINIPath)
		}

		fmt.Println("grafana-config: resolved to", b.GrafanaINIPath)
		return nil
	}

	// check if custom.ini in project root
	dirIni := path.Join(b.ProjectRoot, "custom.ini")
	exists, _ := utils.PathExists(dirIni)
	if exists {
		b.GrafanaINIPath = dirIni
		fmt.Println("grafana-config: custom.ini found. using", b.GrafanaINIPath)
		return nil
	}

	fmt.Println("grafana-config: no config specified. using defaults")
	return nil
}
