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

// DefaultLibraryName derives the library name for PHP purely from the API path.
// It strips the Product Area namespace from the library name, unless the remaining name is generic.
// E.g., "google/cloud/speech/v2" -> "speech"
// E.g., "google/identity/accesscontextmanager/v1" -> "accesscontextmanager"
// E.g., "google/datastore/admin/v1" -> "datastore-admin"
func DefaultLibraryName(apiPath string) string {
	if serviceconfig.ExtractVersion(apiPath) != "" {
		apiPath = path.Dir(apiPath)
	}

	parts := strings.Split(apiPath, "/")
	if len(parts) >= 3 && parts[0] == "google" {
		serviceName := parts[len(parts)-1]
		if serviceName == "admin" || serviceName == "type" {
			// Do not strip the Product Area (PA) for generic services to prevent collisions.
			// E.g., "google/datastore/admin" -> "datastore/admin"
			// E.g., "google/geo/type" -> "geo/type"
			apiPath = strings.TrimPrefix(apiPath, "google/")
		} else {
			// Strip the PA segment unconditionally (e.g., cloud, identity) by removing parts[1].
			// E.g., "google/cloud/speech" -> "speech"
			// E.g., "google/identity/accesscontextmanager" -> "accesscontextmanager"
			apiPath = strings.TrimPrefix(apiPath, "google/"+parts[1]+"/")
		}
	} else {
		apiPath = strings.TrimPrefix(apiPath, "google/")
	}

	apiPath = strings.ReplaceAll(apiPath, "/", "-")
	return strings.ToLower(apiPath)
}

// Add populates PHP-specific default configuration for all APIs in the library.
func Add(lib *config.Library) *config.Library {
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
