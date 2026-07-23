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

package swift

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// LibraryName returns the Swift library (and module) name for the API.
func LibraryName(api *api.API, swiftCfg *config.SwiftPackage) string {
	// TODO(https://github.com/googleapis/librarian/issues/6229) - use
	// a better default.
	parts := strings.Split(api.PackageName, ".")
	for i, p := range parts {
		parts[i] = strcase.ToCamel(p)
	}
	result := strings.Join(parts, "")
	if strings.HasPrefix(result, "Google") {
		return result
	}
	return "Google" + result
}
