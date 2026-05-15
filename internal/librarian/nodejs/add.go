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

package nodejs

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// FindExistingLibraryForNewAPI attempts to find an existing library that should
// contain the given new API path. The rule is currently for matching against
// a candidate library is currently as simple as "if any API path within the
// library matches the given API path, having removed the version from both of
// them, the API should be added to that library". If no such library is found,
// nil is returned.
func FindExistingLibraryForNewAPI(libraries []*config.Library, apiPath string) *config.Library {
	versionlessApiPath := versionless(apiPath)
	for _, lib := range libraries {
		for _, api := range lib.APIs {
			if versionless(api.Path) == versionlessApiPath {
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
