package id

import (
	"fmt"
	"time"
)

// GenRunId returns an unique id for the run from the current time and the test type
// format: {test type}-{year}{day of year}-{hour}{min}{second}
// Example load-2024123-140035
func GenRunId(time time.Time, testType string) string {
	return fmt.Sprintf("%s-%d%d-%d%d%d",
		testType,
		time.Year(),
		time.YearDay(),
		time.Hour(),
		time.Minute(),
		time.Second(),
	)
}
