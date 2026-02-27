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
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	repoMetadataReleaseLevelStable  = "stable"
	repoMetadataReleaseLevelPreview = "preview"
)

func generateRepoMetadata(api *serviceconfig.API, library *config.Library) error {
	goAPI := findGoAPI(library, api.Path)
	if goAPI == nil {
		return errGoAPINotFound
	}
	level, err := metadataReleaseLevel(api, library)
	if err != nil {
		return err
	}
	metadata := &repometadata.RepoMetadata{
		APIShortname:        api.ShortName,
		ClientDocumentation: clientDocURL(goAPI.ImportPath),
		ClientLibraryType:   "generated",
		Description:         api.Title,
		DistributionName:    distributionName(goAPI.ImportPath),
		Language:            "go",
		LibraryType:         repometadata.GAPICAutoLibraryType,
		ReleaseLevel:        level,
	}
	return metadata.Write(filepath.Join(library.Output, goAPI.ImportPath, goAPI.VersionSuffix))
}

// clientDocURL builds the client documentation URL for Go SDK.
func clientDocURL(importPath string) string {
	idx := strings.Index(importPath, "/")

	return fmt.Sprintf("https://cloud.google.com/go/docs/reference/cloud.google.com/go/%s/latest/%s", importPath[:idx], importPath[idx+1:])
}

// distributionName builds the distribution name for Go SDK.
func distributionName(importPath string) string {
	return fmt.Sprintf("cloud.google.com/go/%s", importPath)
}

func metadataReleaseLevel(api *serviceconfig.API, library *config.Library) (string, error) {
	apiReleaseLevel, err := releaseLevel(api.Path, library.Version)
	if err != nil {
		return "", err
	}
	switch apiReleaseLevel {
	case releaseLevelGA:
		return repoMetadataReleaseLevelStable, nil
	default:
		return repoMetadataReleaseLevelPreview, nil
	}
}
