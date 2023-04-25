package utils

import (
	"encoding/json"
	"os"

	"github.com/magefile/mage/sh"
)

// Get working directory
func GetWorkdir() string {
	// Use the Getwd function to get the current working directory
	dir, err := os.Getwd()

	// If there was an error, return it
	if err != nil {
		panic(err)
	}

	// Otherwise, return the current working directory
	return dir
}

// GoEnvInfo returns map of go build environment variables
// More reliable than using uname across operating systems
// This is intended to be used at boot thus we panic if there is an error
func GetCompilerEnvInfo() map[string]string {
	env := make(map[string]string)

	envJson, err := sh.Output("go", "env", "--json")
	if err != nil {

		panic(err)
	}

	err = json.Unmarshal([]byte(envJson), &env)
	if err != nil {
		panic(err)
	}

	return env
}
