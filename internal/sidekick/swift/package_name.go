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
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// PackageName returns the Swift package name for the API.
func PackageName(api *api.API) string {
	return strings.ReplaceAll(api.PackageName, ".", "-")
}

// PackageRepoName derives the split repository name for a library.
//
// When published, libraries are published to standalone repositories
// named with a "swift-" prefix (e.g. "swift-google-cloud-secretmanager-v1").
func PackageRepoName(outdir string, library *config.Library, packageName string) string {
	if outdir != "" {
		base := filepath.Base(outdir)
		if strings.HasPrefix(base, "swift-") {
			return base
		}
	}
	if library != nil && library.Name != "" {
		if strings.HasPrefix(library.Name, "swift-") {
			return library.Name
		}
		return "swift-" + library.Name
	}
	if strings.HasPrefix(packageName, "swift-") {
		return packageName
	}
	return "swift-" + packageName
}
