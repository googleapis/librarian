package utils

import (
	"os"
	"path/filepath"
)

const ENV_VAR_DIRECTORY = "env"
const ENV_VARS_FILENAME = "env_vars.txt"
const RELEASE_ID_ENV_VAR_NAME = "release_id"

// WriteToFile writes the content to a file in the specified filePath.
// It creates the file if it does not exist and truncates it if it does.
func WriteToFile(filePath string, content string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	return writeContentToFile(*file, content)
}

// CreateFile creates a file with the specified name and content in the given directory.
// It will truncate the file if it already exists.
func CreateAndWriteToFile(fileDirectory, fileName string, content string) error {
	path := filepath.Join(fileDirectory, fileName)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	return writeContentToFile(*file, content)
}

func writeContentToFile(file os.File, content string) error {
	_, err := file.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}
