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

package ruby

import (
	"path"
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// FindConfiguredLibrary returns the library in libraries that configures
// apiPath, or nil if there is none. It only reads libraries.
//
// A wrapper library lists the same API path as the versioned library it wraps,
// so a versioned path, such as google/cloud/secretmanager/v1, returns the
// library that is not a wrapper, such as google-cloud-secret_manager-v1. An
// unversioned path, such as google/cloud/secretmanager, returns the wrapper
// library that lists a version of it, such as google-cloud-secret_manager.
func FindConfiguredLibrary(libraries []*config.Library, apiPath string) *config.Library {
	for _, library := range libraries {
		if isWrapperLibrary(library) {
			continue
		}
		if slices.ContainsFunc(library.APIs, func(api *config.API) bool { return api.Path == apiPath }) {
			return library
		}
	}
	if serviceconfig.ExtractVersion(apiPath) != "" {
		return nil
	}
	for _, library := range libraries {
		if !isWrapperLibrary(library) {
			continue
		}
		if slices.ContainsFunc(library.APIs, func(api *config.API) bool { return path.Dir(api.Path) == apiPath }) {
			return library
		}
	}
	return nil
}
