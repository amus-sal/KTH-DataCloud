package commandhandler

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Handle(path string) {

	dest, err := unzip(path, "/usr/files")
	if err != nil {
		fmt.Println(err)
	}
	f, err := os.Create("/tmp/output.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(dest)

	if err != nil {
		log.Fatal(err)
	}
}

func unzip(source, destination string) (string, error) {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return "", err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	err = unzipFile(reader.File[0], destination)
	if err != nil {
		return "", err
	}

	return destination, nil
}

func unzipFile(f *zip.File, destination string) (string, error) {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return filePath, fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return filePath, err
		}
		return filePath, nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return filePath, err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return filePath, err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return filePath, err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return filePath, err
	}
	return filePath, nil
}
