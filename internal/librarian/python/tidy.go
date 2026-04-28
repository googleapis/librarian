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

package python

import (
	"fmt"
	"slices"

	"github.com/googleapis/librarian/internal/config"
)

// Tidy tidies configuration for a library.
func Tidy(lib *config.Library) *config.Library {
	for _, api := range lib.APIs {
		lib = tidyAPI(lib, api)
	}
	return lib
}

// tidyAPI removes redundant OptArgsByAPI values for a single API.
func tidyAPI(lib *config.Library, api *config.API) *config.Library {
	if isProtoOnly(api, lib) || lib.Python == nil {
		return lib
	}
	pythonPackage := lib.Python
	if pythonPackage.OptArgsByAPI == nil {
		return lib
	}
	options, ok := pythonPackage.OptArgsByAPI[api.Path]
	if !ok {
		return lib
	}
	options = deleteMatchingOption(options, gapicNamespaceOption, deriveGAPICNamespace(api.Path))
	options = deleteMatchingOption(options, gapicNameOption, deriveGAPICName(api.Path))
	options = deleteMatchingOption(options, warehousePackageNameOption, lib.Name)
	if len(options) == 0 {
		delete(pythonPackage.OptArgsByAPI, api.Path)
	} else {
		pythonPackage.OptArgsByAPI[api.Path] = options
	}
	if len(pythonPackage.OptArgsByAPI) == 0 {
		pythonPackage.OptArgsByAPI = nil
	}
	return lib
}

// deleteMatchingOptions accepts a slice of original options, the name of an
// option, and the value that can be derived, and returns a slice which is
// contains the original options, without the specified option if it exists
// with the derived value.
func deleteMatchingOption(options []string, optionName, derivedValue string) []string {
	expectedOption := fmt.Sprintf("%s=%s", optionName, derivedValue)
	return slices.DeleteFunc(options, func(option string) bool {
		return option == expectedOption
	})
}
