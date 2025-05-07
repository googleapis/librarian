// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"io"
	"os"
)

func ReadAllBytesFromFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// AppendToFile writes the content to a file in the specified filePath.
// It creates the file if it does not exist, otherwise it appends to existing file.
func AppendToFile(filePath string, content string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeContentToFile(*file, content)
}

// CreateAndWriteToFile creates a file with the specified name and content.
// It will truncate the file if it already exists.
func CreateAndWriteToFile(filePath string, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeContentToFile(*file, content)
}

// CreateAndWriteBytesToFile creates a file with the specified name and content.
// It will truncate the file if it already exists.
func CreateAndWriteBytesToFile(filePath string, content []byte) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeBytesToFile(*file, content)
}

func writeContentToFile(file os.File, content string) error {
	_, err := file.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func writeBytesToFile(file os.File, content []byte) error {
	_, err := file.Write(content)
	if err != nil {
		return err
	}
	return nil
}
