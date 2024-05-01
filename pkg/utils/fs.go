package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Get working directory
func Getwd() string {
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
		return fmt.Errorf("utils: changing directory <%s>: %w", operationDir, err)
	}

	// TODO find a more graceful way to log error and alert
	defer func() {
		if err := os.Chdir(workdir); err != nil {
			panic(err)
		}
	}()

	return fn()
}

// Do function in a directory
func ExecuteInDir(targetDir string, fn func() error) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current work directory %w", err)
	}

	// build the tests
	return DoInDir(workDir, targetDir, fn)
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

// Walks a directory getting a list of all files with matching extensions
// ext must include . like .js
func GlobByExtension(dir string, exts...string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if slices.Contains(exts, filepath.Ext(path)) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// Walks a directory getting a list of all files that match a given prefix
func GlobByPrefix(dir string, prefix string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		base := filepath.Base(path)
		if strings.HasPrefix(base, prefix) {
			files = append(files, base)
		}
		return nil
	})

	return files, err
}

func List(path string) ([]string, error) {
	fileList := []string{}

	files, err := os.ReadDir(path)
	if err != nil {
		return fileList, err
	}

	for _, file := range files {
		if !file.Type().IsRegular() {
			fileList = append(fileList, filepath.Join(path, file.Name()))
		}
	}

	return fileList, nil
}

// rm -rf path
// returns nil if file does not exist
func Rm(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}

	return os.Remove(path)
}

// cp -r src dest
// keeps permissions in-tact
func Cp(src, dest string) error {
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the new file or directory path in the destination
		relativePath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relativePath)

		if info.IsDir() {
			// Create the corresponding directory in the destination
			err = os.MkdirAll(destPath, info.Mode())
			if err != nil {
				return err
			}
		} else {
			// Copy the file from source to destination
			err = copyFile(path, destPath, info.Mode())
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func copyFile(sourcePath, destPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	err = os.Chmod(destPath, mode)
	if err != nil {
		return err
	}

	return nil
}

// cp src dst
//func Cp(src, dst string) error {
//  source, err := os.Open(src)
//  if err != nil {
//    return fmt.Errorf("failed to open source file: %w", err)
//  }
//  defer source.Close()

//  destination, err := os.Create(dst)
//  if err != nil {
//    return fmt.Errorf("failed to create destination file: %w", err)
//  }
//  defer destination.Close()

//  _, err = io.Copy(destination, source)
//  if err != nil {
//    return fmt.Errorf("failed to copy file: %w", err)
//  }

//  return nil
//}
