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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	gapicNamespaceOption = "python-gapic-namespace"
	gapicNameOption      = "python-gapic-name"
)

func gapicNamespace(apiPath string, lib *config.Library) string {
	options := apiOptions(apiPath, lib)
	namespace, ok := findOption(options, gapicNamespaceOption)
	if !ok {
		namespace = deriveGAPICNamespace(apiPath)
	}
	return namespace
}

// deriveGAPICNamespace derives the value to pass as python-gapic-namespace when
// it's not specified explicitly. This is the first two components of the API
// path (excluding any trailing version), dot-separated.
func deriveGAPICNamespace(path string) string {
	version := serviceconfig.ExtractVersion(path)
	if version != "" {
		path = strings.TrimSuffix(path, "/"+version)
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return path
	}
	return parts[0] + "." + parts[1]
}

func gapicName(apiPath string, lib *config.Library) string {
	options := apiOptions(apiPath, lib)
	name, ok := findOption(options, gapicNameOption)
	if !ok {
		name = deriveGAPICName(apiPath)
	}
	return name
}

// deriveGAPICName derives the value to pass as python-gapic-name when it's not
// specified explicitly. This is the path, without the leading namespace (after
// replacing dots with slashes), and without any version suffix, and then
// replacing slashes with underscores. Example:
// a path of google/cloud/foo/bar/v1 would have a GAPIC name of "foo_bar".
func deriveGAPICName(path string) string {
	version := serviceconfig.ExtractVersion(path)
	if version != "" {
		path = strings.TrimSuffix(path, version)
	}
	derivedNamespace := deriveGAPICNamespace(path)
	path = strings.TrimPrefix(path, strings.ReplaceAll(derivedNamespace, ".", "/"))
	path = strings.Trim(path, "/")
	return strings.ReplaceAll(path, "/", "_")
}

// findOption finds the value of a named option within a list of name=value
// strings. If the option isn't found, an empty string is returned. The second
// value indicates whether the option was found or not.
func findOption(options []string, name string) (string, bool) {
	prefix := name + "="
	for _, candidate := range options {
		if after, ok := strings.CutPrefix(candidate, prefix); ok {
			return after, true
		}
	}
	return "", false
}

func apiOptions(apiPath string, lib *config.Library) []string {
	if lib.Python != nil {
		return lib.Python.OptArgsByAPI[apiPath]
	}
	return nil
}
