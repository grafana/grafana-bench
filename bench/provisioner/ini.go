package provisioner

import (
	"context"
	"fmt"

	"github.com/magefile/mage/sh"
)

// This should all be moved to the build package when it's created

// Gets the name of an INI file for build of grafana
func IniFilename(grafanaRevision string) string {
	return fmt.Sprintf("%s_defaults.ini", grafanaRevision)
}

// Downloads defaults.ini for a given build of Grafana to specified directory
func GetBuildINI(ctx context.Context, grafanaRevision, destination string) error {
	// get the ini for that commit of grafana if it doesn't exist
	// takes 7 chars to full commit hash
	iniUrl := fmt.Sprintf("https://raw.githubusercontent.com/grafana/grafana/%s/conf/defaults.ini", grafanaRevision)
	if err := sh.RunV("curl", iniUrl, "-o", destination); err != nil {
		return err
	}

	return nil
}
