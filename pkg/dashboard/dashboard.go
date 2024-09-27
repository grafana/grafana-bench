// Package dashboard implements functions to deal with bench dashboards
package dashboard

import (
	"bytes"
	"fmt"
	"html/template"
)

// RenderDashboardURL takes a URL template and the run identifier and returns the URL 
// to the run's dashboard.
// Example: http://mygrafana.com/b/?var-SuiteRunID={suiteRun}
// 
// NOTE: This functionality may be deprecated in the future.
func RenderDashboardURL(templateURL string, runIdentifier string) (string, error) {
	if templateURL == "" {
		return "", fmt.Errorf("URL template is empty")
	}

	template, err := template.New("dashboard").Parse(templateURL)
	if err != nil {
		return "", fmt.Errorf("error parsing template %w", err)
	}

	// substitution variables
	// TODO: define more substitution variables
	vars := struct {
		SuiteRun string
	}{
		SuiteRun: runIdentifier,
	}

	dashboardURL := bytes.Buffer{}
	err = template.Execute(&dashboardURL, vars)
	if err != nil {
		return "", fmt.Errorf("invalid template substitution: %w", err)
	}

	return dashboardURL.String(), nil
}