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
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
)

func generateRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir string) (*repometadata.RepoMetadata, error) {
	metadata, err := repometadata.FromLibrary(cfg, library, googleapisDir)
	if err != nil {
		return nil, err
	}
	metadata.DistributionName = derivePackageName(library)
	metadata.LibraryType = repometadata.GAPICAutoLibraryType
	metadata.DefaultVersion = resolveDefaultVersion(library)

	if pkgSuffix, ok := strings.CutPrefix(metadata.DistributionName, "@google-cloud/"); ok {
		metadata.ClientDocumentation = fmt.Sprintf("https://cloud.google.com/nodejs/docs/reference/%s/latest", pkgSuffix)
	}

	if library.Nodejs != nil {
		if library.Nodejs.ClientDocumentationOverride != "" {
			metadata.ClientDocumentation = library.Nodejs.ClientDocumentationOverride
		}
		if library.Nodejs.MetadataNameOverride != "" {
			metadata.Name = library.Nodejs.MetadataNameOverride
		}
		if library.Nodejs.NamePrettyOverride != "" {
			metadata.NamePretty = library.Nodejs.NamePrettyOverride
		}
	}

	if strings.HasPrefix(metadata.ProductDocumentation, "https://cloud.google.com/") {
		trimmed := strings.TrimSuffix(metadata.ProductDocumentation, "/")
		if !strings.HasSuffix(trimmed, "/docs") {
			metadata.ProductDocumentation = trimmed + "/docs"
		}
	}
	return metadata, nil
}
