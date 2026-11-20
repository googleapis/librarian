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
// Ruby needs special handling because a wrapper library (e.g.
// google-cloud-secret_manager) lists the same versioned API path as the
// versioned library it wraps (e.g. google-cloud-secret_manager-v1), often
// appearing first in librarian.yaml. A plain first-match lookup would return
// the wrapper for versioned paths and miss it for unversioned ones. Instead,
// versioned paths match only non-wrapper libraries, and unversioned paths match
// a wrapper listing a version of that path.
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
		if slices.ContainsFunc(library.APIs, func(api *config.API) bool {
			return api.Path != "" && path.Dir(api.Path) == apiPath
		}) {
			return library
		}
	}
	return nil
}
