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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Fill populates empty Go-specific fields from the api path.
// Non-default configurations read from librarian.yaml will NOT be overridden.
func Fill(library *config.Library) *config.Library {
	if library.Go == nil {
		library.Go = &config.GoModule{}
	}
	var goAPIs []*config.GoAPI
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if !strings.HasPrefix(api.Path, "google/cloud/") {
			// Do nothing for non cloud API.
			if goAPI != nil {
				goAPIs = append(goAPIs, goAPI)
			}
			continue
		}
		if goAPI == nil {
			goAPI = &config.GoAPI{
				Path: api.Path,
			}
		}
		importPath, clientDir := defaultImportPathAndClientDir(api.Path)
		if goAPI.ImportPath == "" {
			goAPI.ImportPath = importPath
		}
		if goAPI.ClientDirectory == "" {
			goAPI.ClientDirectory = clientDir
		}
		goAPIs = append(goAPIs, goAPI)
	}
	library.Go.GoAPIs = goAPIs

	return library
}

func findGoAPI(library *config.Library, apiPath string) *config.GoAPI {
	if library.Go == nil {
		return nil
	}
	for _, ga := range library.Go.GoAPIs {
		if ga.Path == apiPath {
			return ga
		}
	}
	return nil
}

// defaultImportPathAndClientDir returns the default Go import path and client directory
// based on the provided API path.
//
// The API path is expected to be either google/cloud/{dir}/{version} or
// google/cloud/{dir}/{nested}/{version}.
func defaultImportPathAndClientDir(apiPath string) (string, string) {
	dirs := strings.Split(apiPath, "/")
	if len(dirs) < 4 {
		return "", ""
	}
	if len(dirs) == 5 {
		return fmt.Sprintf("%s/%s", dirs[2], dirs[3]), dirs[3]
	}
	return dirs[2], ""
}
