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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/snippetmetadata"
)

var (
	internalVersionFile = filepath.Join("internal", "version.go")
	versionRegex        = regexp.MustCompile(`(const Version = ")([^"]*)(")`)
)

// Bump updates the version number in the library with the given output
// directory.
func Bump(library *config.Library, output, version string) error {
	if err := bumpInternalVersion(library, output, version); err != nil {
		return err
	}
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("could not find Go API associated with %s: %w", api.Path, errGoAPINotFound)
		}
		snippetDir := snippetDirectory(output, goAPI.ImportPath)
		if err := snippetmetadata.UpdateAllLibraryVersions(snippetDir, version); err != nil {
			return err
		}
	}
	return nil
}

func HasChanges(library *config.Library, filesChanged []string) bool {
	inclusion := filepath.Clean(filepath.Join(library.Output, library.Name))
	// Append file separator to avoid a library name is another library name's prefix.
	if !strings.HasSuffix(inclusion, string(filepath.Separator)) {
		inclusion += string(filepath.Separator)
	}
	var exclusion string
	if library.Go != nil && library.Go.NestedModule != "" {
		exclusion = filepath.Clean(filepath.Join(library.Output, library.Name, library.Go.NestedModule))
		if !strings.HasSuffix(exclusion, string(filepath.Separator)) {
			exclusion += string(filepath.Separator)
		}
	}
	for _, file := range filesChanged {
		if strings.HasPrefix(file, inclusion) {
			if exclusion != "" && strings.HasPrefix(file, exclusion) {
				continue
			}
			return true
		}
	}
	return false
}

func bumpInternalVersion(library *config.Library, output, version string) error {
	versionFilePath := filepath.Join(output, library.Name, internalVersionFile)
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
