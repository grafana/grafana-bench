package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CompressFolder(sourceFolder string, tarGzFileName string) error {
	tarGzFile, err := os.Create(tarGzFileName)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(sourceFolder, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(sourceFolder, filePath)
		if err != nil {
			return err
		}

		header.Name = relativePath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fileInfo.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Folder %s compressed to %s\n", sourceFolder, tarGzFileName)
	return nil
}
