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

package repometadata

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

func buildGoMetadata(metadata *RepoMetadata, description string) *RepoMetadata {
	metadata.ClientLibraryType = "generated"
	metadata.Description = description
	return metadata
}

// goClientDocURL builds the client documentation URL for Go SDK.
func goClientDocURL(library *config.Library, apiPath string) string {
	suffix := fmt.Sprintf("api%s", filepath.Base(apiPath))
	clientDir := clientDirectory(library, apiPath)
	if clientDir != "" {
		suffix = path.Join(clientDir, suffix)
	}
	return fmt.Sprintf("https://cloud.google.com/go/docs/reference/cloud.google.com/go/%s/latest/%s", library.Name, suffix)
}

// goDistributionName builds the distribution name for Go SDK.
func goDistributionName(library *config.Library, apiPath, serviceName string) string {
	version := filepath.Base(apiPath)
	clientDir := clientDirectory(library, apiPath)
	if clientDir != "" {
		serviceName = fmt.Sprintf("%s/%s", serviceName, clientDir)
	}
	return fmt.Sprintf("cloud.google.com/go/%s/api%s", serviceName, version)
}

// TODO(https://github.com/googleapis/librarian/issues/4090): remove this function to reduce code duplication.
func clientDirectory(library *config.Library, apiPath string) string {
	if library.Go == nil || library.Go.GoAPIs == nil {
		return ""
	}
	for _, api := range library.Go.GoAPIs {
		if api.Path == apiPath {
			return api.ClientDirectory
		}
	}
	return ""
}
