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

package python

import (
	"errors"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// defaultVersion is the first version used for a new library.
// This is set on the initial `librarian add` for a new API.
const defaultVersion = "0.0.0"

var errNewLibraryMustHaveOneAPI = errors.New("a newly added library must have exactly one API")

// Add initializes a new Python library with default values.
func Add(lib *config.Library) (*config.Library, error) {
	lib.Version = defaultVersion
	if len(lib.APIs) != 1 {
		return nil, errNewLibraryMustHaveOneAPI
	}
	if packageDefaultVersion := serviceconfig.ExtractVersion(lib.APIs[0].Path); packageDefaultVersion != "" {
		lib.Python = &config.PythonPackage{
			DefaultVersion: packageDefaultVersion,
		}
	}
	return lib, nil
}
