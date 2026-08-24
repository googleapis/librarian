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

package php

import (
	"path"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const defaultVersion = "0.0.0"

// DefaultLibraryName derives the library name for PHP purely from the API path.
// E.g., "google/cloud/speech/v2" -> "speech"
// E.g., "google/cloud/security/privateca/v1" -> "security-privateca".
func DefaultLibraryName(apiPath string) string {
	apiPath = strings.TrimPrefix(apiPath, "google/cloud/")
	apiPath = strings.TrimPrefix(apiPath, "google/")
	if serviceconfig.ExtractVersion(apiPath) != "" {
		apiPath = path.Dir(apiPath)
	}
	apiPath = strings.ReplaceAll(apiPath, "/", "-")
	return strings.ToLower(apiPath)
}

// Add populates PHP-specific default configuration for all APIs in the library.
func Add(lib *config.Library) *config.Library {
	if lib.Version == "" {
		lib.Version = defaultVersion
	}
	for _, api := range lib.APIs {
		if api.PHP == nil {
			api.PHP = &config.PHPAPI{}
		}
		if api.PHP.StagingSubdir == "" {
			api.PHP.StagingSubdir = serviceconfig.ExtractVersion(api.Path)
		}
	}
	return lib
}
