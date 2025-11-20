// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package legacyconfig

import (
	"log/slog"
	"strings"
)

const defaultTagFormat = "{id}-{version}"

// DetermineTagFormat finds the tag_format config given a library ID.
func DetermineTagFormat(libraryID string, libraryState *LibraryState, librarianConfig *LibrarianConfig) string {
	// Order of preference:
	// 1. per-library from config.yaml
	// 2. top-level from config.yaml
	// 3. per-library from state.yaml (deprecated)
	if librarianConfig != nil {
		// prefer per-library config
		libraryConfig := librarianConfig.LibraryConfigFor(libraryID)
		if libraryConfig != nil && libraryConfig.TagFormat != "" {
			return libraryConfig.TagFormat
		}
		// top-level from config
		if librarianConfig.TagFormat != "" {
			return librarianConfig.TagFormat
		}
	}

	if libraryState != nil {
		if libraryState.TagFormat != "" {
			return libraryState.TagFormat
		}
	}
	slog.Warn("library did not configure tag_format, using default", "libraryID", libraryID, "format", defaultTagFormat)
	return defaultTagFormat
}

// FormatTag returns the git tag for a given library version.
func FormatTag(tagFormat string, libraryID string, version string) string {
	if tagFormat == "" {
		slog.Warn("not tag format specified, using default", "format", defaultTagFormat)
		tagFormat = defaultTagFormat
	}
	r := strings.NewReplacer("{id}", libraryID, "{version}", version)
	return r.Replace(tagFormat)
}
