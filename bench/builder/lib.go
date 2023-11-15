package builder

import (
	"strings"
)

// validate arch string linux/amd64
func validateArch(archstring string) bool {
	parts := strings.Split(archstring, "/")
	if len(parts) != 2 {
		return false
	}

	os := parts[0]
	if os != "linux" && os != "darwin" && os != "windows" {
		return false
	}

	arch := parts[1]
	if arch != "amd64" && arch != "arm64" {
		return false
	}

	return true
}
