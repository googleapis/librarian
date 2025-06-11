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

package librarian

import (
	"io"
	"os"
)

func readAllBytesFromFile(filePath string) (_ []byte, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()
	return io.ReadAll(file)
}

// appendToFile writes the content to a file in the specified filePath.
// It creates the file if it does not exist, otherwise it appends to existing file.
func appendToFile(filePath string, content string) (err error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = file.WriteString(content)
	return err
}

// createAndWriteToFile creates a file with the specified name and content.
// It will truncate the file if it already exists.
func createAndWriteToFile(filePath string, content string) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = file.WriteString(content)
	return err
}

// createAndWriteBytesToFile creates a file with the specified name and content.
// It will truncate the file if it already exists.
func createAndWriteBytesToFile(filePath string, content []byte) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = file.Write(content)
	return err
}
