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

package main

import (
	"github.com/googleapis/librarian/internal/config"
)

// buildPythonLibraries builds a set of librarian libraries from legacylibrarian
// libraries and the googleapis directory used to find settings in service
// config files, BUILD.bazel files etc.
func buildPythonLibraries(input *MigrationInput, googleapisDir string) ([]*config.Library, error) {
	var libraries []*config.Library
	// No need to use legacyconfig.LibraryConfig - the only thing in
	// the python config is a single global file entry.

	for _, libState := range input.librarianState.Libraries {
		library := &config.Library{}
		library.Name = libState.ID
		library.Version = libState.Version
		if libState.APIs != nil {
			library.APIs = toAPIs(libState.APIs)
		}
		libraries = append(libraries, library)
	}
	return libraries, nil
}
