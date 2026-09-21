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
	"path"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// ReleasePleaseExtraFiles returns the extra-files tracked by release-please for Python libraries.
func ReleasePleaseExtraFiles(lib *config.Library) []any {
	var extraFiles []any
	addedVersionless := make(map[string]bool)
	for _, api := range lib.APIs {
		flattenedPath := flattenNestedPath(api.Path, lib)
		if !addedVersionless[flattenedPath] {
			addedVersionless[flattenedPath] = true
			extraFiles = append(extraFiles, flattenedPath+"/gapic_version.py")
		}
		version := serviceconfig.ExtractVersion(api.Path)
		if version != "" {
			extraFiles = append(extraFiles, flattenedPath+"_"+version+"/gapic_version.py")
		}
		protoPackage := strings.ReplaceAll(api.Path, "/", ".")
		// https://github.com/googleapis/release-please/blob/main/docs/customizing.md#updating-arbitrary-files
		snippetMetadata := map[string]any{
			"jsonpath": "$.clientLibrary.version",
			"path":     "samples/generated_samples/snippet_metadata_" + protoPackage + ".json",
			"type":     "json",
		}
		extraFiles = append(extraFiles, snippetMetadata)
	}
	return extraFiles
}

// flattenNestedPath flattens nested paths in apiPath, specifically for non-cloud API prefixes.
// For example, google/shopping/merchant/inventories becomes google/shopping/merchant_inventories.
func flattenNestedPath(apiPath string, lib *config.Library) string {
	namespace := strings.ReplaceAll(gapicNamespace(apiPath, lib), ".", "/")
	name := gapicName(apiPath, lib)
	return path.Join(namespace, name)
}
