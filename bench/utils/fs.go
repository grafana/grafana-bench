package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Copy file from one place to another
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

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

// Walks a directory getting a list of all files.
// ext must include . like .js
func Glob(dir string, ext string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
