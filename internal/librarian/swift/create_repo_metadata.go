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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sources"
)

func createRepoMetadata(cfg *config.Config, library *config.Library, src *sources.Sources) (*repometadata.RepoMetadata, error) {
	metadata, err := repometadata.FromLibrary(cfg, library, src.Googleapis)
	if err != nil {
		return nil, err
	}
	// Language specific data.
	metadata.ClientDocumentation = fmt.Sprintf(
		"https://swiftpackageindex.com/googleapis/%s/documentation", library.Name)
	metadata.LibraryType = repometadata.GAPICAutoLibraryType

	return metadata, nil
}

func needsRepoMetadata(model *api.API, library *config.Library) bool {
	if len(model.Services) == 0 {
		return false
	}
	if len(library.APIs) == 0 {
		return false
	}
	if library.APIs[0].Path == repometadata.ShowcasePath {
		return false
	}
	return true
}
