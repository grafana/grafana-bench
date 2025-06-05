package format

import (
	"fmt"
	"time"
)

// PrettyMS formats a duration in ms
func PrettyMS(duration time.Duration) string {
	return fmt.Sprintf("%dms", duration.Milliseconds())
}