package env

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Gets the architecture of the machine running Bench
func GetLocalArch() string {
	return fmt.Sprintf("%s/%s", strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH))
}

// Get environment variable or use default value
func EnvOrDefault(environmentVarName, defaultValue string) string {
	v := os.Getenv(environmentVarName)
	if v == "" {
		return defaultValue
	}

	return v
}

// Get boolean environment variable. panics if there's an issue with conversion
func EnvOrDefaultBool(environmentVarName, defaultValue string) bool {
	bool, err := strconv.ParseBool(EnvOrDefault(environmentVarName, defaultValue))
	if err != nil {
		panic(fmt.Sprintf("error reading bool env variable %s: %s", environmentVarName, err))
	}
	return bool
}
