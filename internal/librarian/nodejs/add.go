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
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const defaultVersion = "0.0.0"

// Add initializes Node.js-specific configuration for a library.
func Add(cfg *config.Config, lib *config.Library) *config.Library {
	lib.Version = defaultVersion
	if cfg != nil {
		lib = fillDefault(lib, cfg.Default)
	}
	apiPath := lib.APIs[0].Path
	if !strings.HasPrefix(apiPath, "google/cloud/") && (lib.Nodejs == nil || lib.Nodejs.PackageName == "") {
		slog.Warn("unrecognized non-cloud API path; please manually configure nodejs.package_name in librarian.yaml", "api", apiPath)
	}
	return lib
}

// fillDefault populates empty Node.js-specific fields in lib from [config.Default],
// specifically from [config.NodejsDefault].
func fillDefault(lib *config.Library, d *config.Default) *config.Library {
	if lib.Nodejs != nil && lib.Nodejs.PackageName != "" {
		return lib
	}
	if d == nil || d.Nodejs == nil || len(d.Nodejs.CustomPackagePrefixes) == 0 {
		return lib
	}
	if pkgName := derivePackageNameFromCustomPackagePrefixes(lib.APIs[0].Path, d.Nodejs.CustomPackagePrefixes); pkgName != "" {
		if lib.Nodejs == nil {
			lib.Nodejs = &config.NodejsPackage{}
		}
		lib.Nodejs.PackageName = pkgName
	}
	return lib
}

func derivePackageNameFromCustomPackagePrefixes(apiPath string, customPackagePrefixes map[string]string) string {
	scope, remainder, ok := serviceconfig.MatchPrefix(apiPath, customPackagePrefixes)
	if !ok {
		return ""
	}
	sub := strings.ReplaceAll(remainder, "/", "-")
	switch {
	case sub == "":
		// Exact leaf API match (e.g. @google-apps/chat).
		return scope
	case strings.Contains(scope, "/"):
		// Scope already has a slash; join with hyphen (e.g. @google/area120-tables).
		return scope + "-" + sub
	default:
		// Standard scope; join with slash (e.g. @google-shopping/accounts).
		return scope + "/" + sub
	}
}

// DefaultLibraryName derives a library name from an API path by stripping
// the version suffix and replacing "/" with "-".
// For example: "google/cloud/secretmanager/v1" ->
// "google-cloud-secretmanager".
func DefaultLibraryName(api string) string {
	slashPath := api
	if serviceconfig.ExtractVersion(api) != "" {
		// Strip version suffix (v1, v1beta2, v2alpha, etc.).
		slashPath = filepath.Dir(api)
	}
	return strings.ReplaceAll(slashPath, "/", "-")
}

// FindExistingLibraryForNewAPI attempts to find an existing library that should
// contain the given new API path. The rule is currently for matching against
// a candidate library is currently as simple as "if any API path within the
// library matches the given API path, having removed the version from both of
// them, the API should be added to that library". If no such library is found,
// nil is returned.
func FindExistingLibraryForNewAPI(libraries []*config.Library, apiPath string) *config.Library {
	versionlessApiPath := versionless(apiPath)
	for _, lib := range libraries {
		for _, api := range lib.APIs {
			if versionless(api.Path) == versionlessApiPath {
				return lib
			}
		}
	}
	return nil
}

// versionless trims the version (if any) from apiPath, leaving any trailing
// slash.
func versionless(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	return strings.TrimSuffix(apiPath, version)
}
