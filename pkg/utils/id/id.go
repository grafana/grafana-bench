package id

import (
	"fmt"
	"strings"
	"time"
)

// Run returns a unique id for the test suite run including stage, suite name, and timestamp
// format: {stage}-{suiteName}-{year}{dayOfYear:03d}-{hour:02d}{min:02d}{sec:02d}
// Example: rrc-grafana-api-tests-tests-2026022-140035
// Example: local-grafana-api-tests-tests-2026022-140035
//
// The suiteName is normalized to remove slashes (replaced with hyphens) for better readability
// and compatibility with various systems.
func Run(stage string, suiteName string, t time.Time) string {
	// Normalize suite name: replace slashes with hyphens
	normalizedSuiteName := strings.ReplaceAll(suiteName, "/", "-")

	return fmt.Sprintf("%s-%s-%d%03d-%02d%02d%02d",
		stage,
		normalizedSuiteName,
		t.Year(),
		t.YearDay(),
		t.Hour(),
		t.Minute(),
		t.Second(),
	)
}
