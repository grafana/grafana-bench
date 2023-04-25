package utils

import "os"

// Do function in a directory
func DoInDir(workdir string, operationDir string, fn func() error) error {
	if err := os.Chdir(operationDir); err != nil {
		return err
	}

	// TODO find a more graceful way to log error and alert
	defer func() {
		if err := os.Chdir(workdir); err != nil {
			panic(err)
		}
	}()

	return fn()
}

// Checks for existence of directory
func PathExists(path string) (bool, error) {
	// Use the Stat function to check if the directory exists
	_, err := os.Stat(path)

	// If there is no error, the path exists
	if err == nil {
		return true, nil
	}

	// If the error is "not exists", the directory does not exist
	if os.IsNotExist(err) {
		return false, nil
	}

	// If the error is not "not exists", return the error
	return false, err
}
