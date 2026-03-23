// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package golang

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/googleapis/librarian/internal/config"
)

var (
	internalVersionFile = filepath.Join("internal", "version.go")
	versionRegex        = regexp.MustCompile(`(const Version = ")([^"]*)(")`)
)

// Bump updates the version number in the library with the given output
// directory.
func Bump(library *config.Library, output, version string) error {
	library, err := Fill(library)
	if err != nil {
		return err
	}
	if err := bumpInternalVersion(output, version); err != nil {
		return err
	}
	return updateSnippetDirectory(library, output, version)
}

func bumpInternalVersion(output, version string) error {
	versionFilePath := filepath.Join(output, internalVersionFile)
	if _, err := os.Stat(versionFilePath); os.IsNotExist(err) {
		return nil
	}

	return findAndReplace(versionFilePath, version)
}

func findAndReplace(path string, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result := versionRegex.ReplaceAllString(string(content), `${1}`+version+`${3}`)
	return os.WriteFile(path, []byte(result), 0644)
}
