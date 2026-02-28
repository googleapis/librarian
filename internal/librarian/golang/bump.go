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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/snippetmetadata"
)

var (
	internalVersionFile = filepath.Join("internal", "version.go")

	versionRegex = regexp.MustCompile(`(const Version = ")([0-9.]+)(")`)
)

// Bump updates the version number in the library with the given output
// directory.
func Bump(library *config.Library, output, version string) error {
	if err := bumpInternalVersion(library, output, version); err != nil {
		return err
	}
	snippetsDir := filepath.Join(output, "internal", "generated", "snippets", library.Name)
	return snippetmetadata.UpdateAllLibraryVersions(snippetsDir, version)
}

func bumpInternalVersion(library *config.Library, output, version string) error {
	libraryDir := filepath.Join(output, library.Name)
	return filepath.WalkDir(libraryDir, func(path string, d fs.DirEntry, err error) error {

		if d.IsDir() {
			if library.Go != nil && d.Name() == library.Go.NestedModule {
				return filepath.SkipDir
			}
		}
		if !strings.HasSuffix(path, internalVersionFile) {
			return nil
		}
		return findAndReplace(path, version)
	})
}

func findAndReplace(path string, version string) error {
	// The internal version.go is small, it should be good to read the file
	// in one go.
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result := versionRegex.ReplaceAllString(string(content), `${1}`+version+`${3}`)
	return os.WriteFile(path, []byte(result), 0644)
}
