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
	"fmt"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// defaultVersion is the first version used for a new library.
// This is set on the initial `librarian add` for a new API.
const defaultVersion = "0.0.0"

var (
	approvedGAPICNamespaces = []string{
		"google.ads",
		"google.apps",
		"google.cloud",
		"google.maps",
		"google.shopping",
	}
	errNewLibraryMustHaveOneAPI = errors.New("a newly added library (in Python) must have exactly one API so that the default version can be populated")
	errNewLibraryBadNamespace   = errors.New("derived GAPIC namespace would not match any approved namespace; consult with the Python team to determine whether the namespace should be approved, or whether GAPIC options should be specified for this API in librarian.yaml. See go/clientlibs-python-registered-namespaces for more details")
)

// Add initializes a new Python library with default values.
func Add(lib *config.Library) (*config.Library, error) {
	lib.Version = defaultVersion
	if len(lib.APIs) != 1 {
		return nil, errNewLibraryMustHaveOneAPI
	}
	apiPath := lib.APIs[0].Path
	if packageDefaultVersion := serviceconfig.ExtractVersion(apiPath); packageDefaultVersion != "" {
		lib.Python = &config.PythonPackage{
			DefaultVersion: packageDefaultVersion,
		}
	}
	namespace := deriveGAPICNamespace(apiPath)
	if !slices.Contains(approvedGAPICNamespaces, namespace) {
		return nil, fmt.Errorf("unapproved namespace %s derived from API path %s: %w", namespace, apiPath, errNewLibraryBadNamespace)
	}
	return lib, nil
}
