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
	"strings"

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
	errNewLibraryMustHaveOneAPI          = errors.New("a newly added library (in Python) must have exactly one API so that the default version can be populated")
	errNewLibraryBadNamespace            = errors.New("derived GAPIC namespace would not match any approved namespace; consult with the Python team to determine whether the namespace should be approved, or whether GAPIC options should be specified for this API in librarian.yaml. See go/clientlibs-python-registered-namespaces for more details")
	errExistingLibraryNoDefaultVersion   = errors.New("new APIs cannot be automatically added to a library without a default version")
	errExistingLibraryCustomGAPICOptions = errors.New("new APIs cannot be automatically added to a library with custom GAPIC options")
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
		return nil, fmt.Errorf("%w: unapproved namespace %s derived from API path %s", errNewLibraryBadNamespace, namespace, apiPath)
	}
	return lib, nil
}

// ValidateNewAPIs validates that new APIs can be added to an existing library.
// Currently this is just a check that there is a default version already, and
// that no existing APIs in the library have custom GAPIC options. Future checks
// may require details of the APIs being added.
func ValidateNewAPIs(lib *config.Library) error {
	if lib.Python == nil || lib.Python.DefaultVersion == "" {
		return errExistingLibraryNoDefaultVersion
	}
	if len(lib.Python.OptArgsByAPI) != 0 {
		return errExistingLibraryCustomGAPICOptions
	}
	return nil
}

// FindExistingLibraryForNewAPI attempts to find an existing library that should
// contain the given new API path. This function uses the concept of a
// "versionless" path, which is just the path with any version removed (so the
// versionless path of google/cloud/xyz/v1 is google/cloud/xyz/; the versionless
// path of google/cloud/xyz/type is still google/cloud/xyz/type). A nil pointer
// is returned if no existing library is suitable for the new API path. The
// function observes the following rules:
//
//  1. If the versionless path of any path in the library is the same as the
//     versionless apiPath, that library is returned.
//  2. If the versionless path of any path in the library is a prefix of apiPath
//     *and* the library contains multiple different versionless paths, that
//     library is returned.
//  3. Otherwise, nil is returned.
func FindExistingLibraryForNewAPI(libraries []*config.Library, apiPath string) *config.Library {
	versionlessApiPath := versionless(apiPath)
	for _, lib := range libraries {
		for _, api := range lib.APIs {
			if versionless(api.Path) == versionlessApiPath {
				return lib
			}
		}
	}
	for _, lib := range libraries {
		set := make(map[string]struct{})
		for _, api := range lib.APIs {
			set[versionless(api.Path)] = struct{}{}
		}
		if len(set) <= 1 {
			continue
		}
		for v := range set {
			if strings.HasPrefix(apiPath, v) {
				return lib
			}
		}
	}
	return nil
}

// versionless trims the version (if any) from apiPath, leaving any trailing
// slash.
func versionless(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	return strings.TrimSuffix(apiPath, version)
}
