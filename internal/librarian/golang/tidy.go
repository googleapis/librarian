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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Tidy tidies the Go-specific configuration for a library by removing default
// values and clearing empty Go module or API entries.
func Tidy(library *config.Library) *config.Library {
	library.Output = strings.TrimSuffix(library.Output, library.Name)
	if library.Go == nil {
		return library
	}
	var goAPIs []*config.GoAPI
	for _, goAPI := range library.Go.GoAPIs {
		importPath, clientPkg := defaultImportPathAndClientPkg(goAPI.Path)
		if goAPI.ImportPath == importPath {
			goAPI.ImportPath = ""
		}
		if goAPI.ClientPackage == clientPkg {
			goAPI.ClientPackage = ""
		}
		if !isEmptyAPI(goAPI) {
			goAPIs = append(goAPIs, goAPI)
		}
	}
	library.Go.GoAPIs = goAPIs
	if isEmptyGoModule(library.Go) {
		library.Go = nil
	}
	return library
}

func isEmptyAPI(goAPI *config.GoAPI) bool {
	return goAPI.ClientPackage == "" &&
		!goAPI.DIREGAPIC &&
		len(goAPI.EnabledGeneratorFeatures) == 0 &&
		goAPI.ImportPath == "" &&
		len(goAPI.NestedProtos) == 0 &&
		!goAPI.NoMetadata &&
		!goAPI.ProtoOnly &&
		goAPI.ProtoPackage == ""
}

func isEmptyGoModule(goModule *config.GoModule) bool {
	return len(goModule.DeleteGenerationOutputPaths) == 0 &&
		len(goModule.GoAPIs) == 0 &&
		goModule.ModulePathVersion == "" &&
		goModule.NestedModule == ""
}
