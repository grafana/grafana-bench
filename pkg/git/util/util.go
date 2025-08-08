package util

import (
	"fmt"
	"os"
)

// ValidateTargetDir validates the target directory exists and is empty
func ValidateTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)

	// if not exists, it's fine, we will create it
	if os.IsNotExist(err) {
		return nil
	}

	// un expected error
	if err != nil {
		return fmt.Errorf("accessing target dir %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target must be a directory")
	}

	empty, err := isEmpty(targetDir)
	if err != nil {
		return err
	}

	if !empty {
		return fmt.Errorf("target dir must be empty")
	}

	return nil
}

func isEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("accessing directory %w", err)
	}

	return len(files) == 0, nil
}