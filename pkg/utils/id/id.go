package id

import (
	"fmt"
	"time"
)

// Run returns an unique id for the run from the pipeline and current time
// format: {trigger}-{year}{day of year}-{hour}{min}{second}
// Example rrc-dev-fast-6-2024123-140035
func Run(trigger string, time time.Time) string {
	return fmt.Sprintf("%s-%d%d-%d%d%d",
		trigger,
		time.Year(),
		time.YearDay(),
		time.Hour(),
		time.Minute(),
		time.Second(),
	)
}

// SuiteRunName returns an unique id for the pipeline
// format: {trigger}-{suite name}-{test type}
// Example rrc-dev-fast-grafana/grafana-api-tests-load
func SuiteRunName(trigger string, suiteName string, testType string) string {
	return fmt.Sprintf("%s-%s-%s", trigger, suiteName, testType)
}
