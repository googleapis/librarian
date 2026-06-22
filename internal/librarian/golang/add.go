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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// defaultVersion is the first version used for a new library.
// This is set on the initial `librarian add` for a new API.
const defaultVersion = "0.0.0"

// Add initializes a Go library with default values.
func Add(lib *config.Library) *config.Library {
	if lib.Version == "" {
		lib.Version = defaultVersion
	}
	for _, api := range lib.APIs {
		addGoAPI(api)
	}
	return lib
}

// addGoAPI initializes Go-specific API configuration when adding a new API.
// It populates the ImportPath and sets ProtoOnly to true if the API path
// is versionless (does not contain a version segment like "v1"). It does
// nothing for versioned API paths.
func addGoAPI(api *config.API) {
	if serviceconfig.ExtractVersion(api.Path) != "" {
		return
	}
	if api.Go != nil {
		return
	}
	importPath := deriveVersionlessImportPath(api.Path)
	api.Go = &config.GoAPI{
		ImportPath: importPath,
		ProtoOnly:  true,
	}
}

func deriveVersionlessImportPath(apiPath string) string {
	apiPath = strings.TrimPrefix(apiPath, "google/cloud/")
	apiPath = strings.TrimPrefix(apiPath, "google/")
	idx := strings.LastIndex(apiPath, "/")
	var leaf string
	if idx == -1 {
		leaf = apiPath
	} else {
		leaf = apiPath[idx+1:]
	}
	return fmt.Sprintf("%s/%spb", apiPath, leaf)
}

// ReleasePleaseExtraFiles returns the extra-files tracked by release-please for Go libraries.
func ReleasePleaseExtraFiles(lib *config.Library) []any {
	var extraFiles []any
	for _, api := range lib.APIs {
		goAPI := api.Go
		if goAPI == nil {
			goAPI = &config.GoAPI{}
		}
		if goAPI.ProtoOnly || goAPI.NoSnippets {
			continue
		}

		importPath := goAPI.ImportPath
		if importPath == "" {
			importPath, _ = defaultImportPathAndClientPkg(api.Path)
		}
		if importPath == "" {
			continue
		}

		clientPath := importPath
		if lib.Go != nil && lib.Go.ModulePathVersion != "" {
			modulePathVersion := "/" + lib.Go.ModulePathVersion
			clientPath = strings.Replace(clientPath, modulePathVersion, "", 1)
		}

		snippetDir := path.Join("examples", strings.TrimPrefix(clientPath, lib.Name+"/"))
		if lib.Go != nil {
			deleted := false
			for _, delPath := range lib.Go.DeleteGenerationOutputPaths {
				if strings.HasPrefix(snippetDir, delPath) {
					deleted = true
					break
				}
			}
			if deleted {
				continue
			}
		}

		protoPackage := goAPI.ProtoPackage
		if protoPackage == "" {
			protoPackage = strings.ReplaceAll(api.Path, "/", ".")
		}

		snippetMetadata := map[string]any{
			"jsonpath": "$.clientLibrary.version",
			"path":     path.Join(snippetDir, "snippet_metadata."+protoPackage+".json"),
			"type":     "json",
		}
		extraFiles = append(extraFiles, snippetMetadata)
	}
	return extraFiles
}
