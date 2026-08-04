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

package swift

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/iancoleman/strcase"
)

const requiresLibraryNameOverrideFormat = `default library name for %s needs override.
Other languages with PascalCase style deviate from the default name for this library,
most likely, that indicates the default library name is not a good choice. Consider
these alternatives and use library_name_override to silence this error:
%s`

// LibraryName returns the Swift library (and module) name for the API.
func LibraryName(api *api.API, swiftCfg *config.SwiftPackage) (string, error) {
	if swiftCfg != nil && swiftCfg.LibraryNameOverride != "" {
		return swiftCfg.LibraryNameOverride, nil
	}
	if api.PackageName == "" {
		return "", fmt.Errorf("API package name must not be empty")
	}
	parts := strings.Split(api.PackageName, ".")
	for i, p := range parts {
		parts[i] = strcase.ToCamel(p)
	}
	libraryName := strings.Join(parts, "")
	alternatives := []struct {
		lang  string
		value string
	}{
		{"C#", strings.ReplaceAll(api.CsharpNamespace, ".", "")},
		{"PHP", strings.ReplaceAll(api.PhpNamespace, "\\", "")},
		{"Ruby", strings.ReplaceAll(api.RubyPackage, "::", "")},
	}
	var messages []string
	for _, alt := range alternatives {
		if alt.value == "" {
			continue
		}
		if alt.value != libraryName {
			messages = append(messages, fmt.Sprintf("%s suggests using %s", alt.lang, alt.value))
		}
	}
	if len(messages) != 0 {
		return "", fmt.Errorf(requiresLibraryNameOverrideFormat, api.PackageName, strings.Join(messages, "\n"))
	}

	return libraryName, nil
}
