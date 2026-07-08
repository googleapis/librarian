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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

// PackageName returns the package name for the API.
func PackageName(api *api.API) string {
	var name string
	if suffix, ok := strings.CutPrefix(api.PackageName, "google.cloud."); ok {
		name = "Cloud" + pascalPackageName(suffix)
	} else if suffix, ok := strings.CutPrefix(api.PackageName, "google."); ok {
		name = pascalPackageName(suffix)
	} else {
		name = pascalPackageName(api.PackageName)
	}
	return "Google" + name
}

func pascalPackageName(packageName string) string {
	parts := strings.Split(packageName, ".")
	var name strings.Builder
	for _, p := range parts {
		name.WriteString(strcase.ToCamel(p))
	}
	return name.String()
}
