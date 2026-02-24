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

package golang

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

type RepoMetadata struct {
	repometadata.RepoMetadata `json:",inline"`
	ClientLibraryType         string `json:"client_library_type,omitempty"`
	Description               string `json:"description,omitempty"`
}

func generateRepoMetadata(api *serviceconfig.API, library *config.Library) error {
	metadata := &RepoMetadata{
		RepoMetadata: repometadata.RepoMetadata{
			APIShortname:        api.ShortName,
			ClientDocumentation: clientDocURL(library, api.Path),
			DistributionName:    distributionName(library, api.Path, api.ShortName),
			Language:            "go",
			LibraryType:         repometadata.GAPICAutoLibraryType,
			ReleaseLevel:        library.ReleaseLevel,
		},
		ClientLibraryType: "generated",
		Description:       repometadata.CleanTitle(api.Title),
	}
	dir, _ := resolveClientPath(library, api.Path)
	return repometadata.Write(metadata, dir)
}

// clientDocURL builds the client documentation URL for Go SDK.
func clientDocURL(library *config.Library, apiPath string) string {
	suffix := fmt.Sprintf("api%s", filepath.Base(apiPath))
	clientDir := clientDirectory(library, apiPath)
	if clientDir != "" {
		suffix = path.Join(clientDir, suffix)
	}
	return fmt.Sprintf("https://cloud.google.com/go/docs/reference/cloud.google.com/go/%s/latest/%s", library.Name, suffix)
}

// distributionName builds the distribution name for Go SDK.
func distributionName(library *config.Library, apiPath, serviceName string) string {
	version := filepath.Base(apiPath)
	clientDir := clientDirectory(library, apiPath)
	if clientDir != "" {
		serviceName = fmt.Sprintf("%s/%s", serviceName, clientDir)
	}
	return fmt.Sprintf("cloud.google.com/go/%s/api%s", serviceName, version)
}
