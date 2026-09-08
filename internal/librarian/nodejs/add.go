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
	"log"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const defaultVersion = "0.0.0"

// Add initializes Node.js-specific configuration for a library.
func Add(cfg *config.Config, lib *config.Library) *config.Library {
	lib.Version = defaultVersion
	var d *config.Default
	if cfg != nil {
		d = cfg.Default
	}
	lib = FillDefaultNodejs(lib, d)
	if len(lib.APIs) > 0 {
		apiPath := lib.APIs[0].Path
		if !strings.HasPrefix(apiPath, "google/cloud/") && (lib.Nodejs == nil || lib.Nodejs.PackageName == "") {
			log.Printf("WARNING: unrecognized non-cloud API path %q. Please manually configure nodejs.package_name in librarian.yaml.", apiPath)
		}
	}
	return lib
}

// FillDefaultNodejs populates empty Node.js-specific fields in lib from [config.Default],
// specifically from [config.NodejsDefault].
func FillDefaultNodejs(lib *config.Library, d *config.Default) *config.Library {
	if lib == nil {
		return nil
	}
	fillPackageNameIfEmpty(lib, d)
	return lib
}

func fillPackageNameIfEmpty(lib *config.Library, d *config.Default) {
	if lib.Nodejs != nil && lib.Nodejs.PackageName != "" {
		return
	}
	if d == nil || d.Nodejs == nil || len(d.Nodejs.CustomScopes) == 0 {
		return
	}
	for _, api := range lib.APIs {
		if pkgName := derivePackageNameFromCustomScopes(api.Path, d.Nodejs.CustomScopes); pkgName != "" {
			if lib.Nodejs == nil {
				lib.Nodejs = &config.NodejsPackage{}
			}
			lib.Nodejs.PackageName = pkgName
			return
		}
	}
}

func derivePackageNameFromCustomScopes(apiPath string, customScopes map[string]string) string {
	if apiPath == "" || len(customScopes) == 0 {
		return ""
	}
	pathWithoutVersion := apiPath
	if v := serviceconfig.ExtractVersion(apiPath); v != "" {
		pathWithoutVersion = strings.TrimSuffix(strings.TrimSuffix(apiPath, v), "/")
	}
	prefixes := make([]string, 0, len(customScopes))
	for p := range customScopes {
		prefixes = append(prefixes, p)
	}
	slices.SortFunc(prefixes, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	for _, prefix := range prefixes {
		normPrefix := strings.Trim(prefix, "/")
		var remainder string
		if pathWithoutVersion == normPrefix {
			remainder = ""
		} else if rem, ok := strings.CutPrefix(pathWithoutVersion, normPrefix+"/"); ok {
			remainder = rem
		} else {
			continue
		}
		scope := strings.TrimSpace(customScopes[prefix])
		if scope == "" {
			continue
		}
		if !strings.HasPrefix(scope, "@") {
			scope = "@" + scope
		}
		if strings.Contains(scope, "/") {
			if remainder == "" {
				return scope
			}
			return fmt.Sprintf("%s-%s", scope, strings.ReplaceAll(remainder, "/", "-"))
		}
		if remainder != "" {
			return fmt.Sprintf("%s/%s", scope, strings.ReplaceAll(remainder, "/", "-"))
		}
		return fmt.Sprintf("%s/%s", scope, filepath.Base(normPrefix))
	}
	return ""
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
