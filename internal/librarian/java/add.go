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

package java

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// knownPrefixes contains API path prefixes to be stripped when deriving a
// library name. The order matters: more specific prefixes must come before
// less specific ones (e.g., "google/cloud/" before "google/").
var knownPrefixes = []string{
	"google/cloud/",
	"google/api/",
	"google/devtools/",
	"google/",
}

const defaultVersion = "0.1.0-SNAPSHOT"

// Add initializes a new Java library with default values.
func Add(lib *config.Library) *config.Library {
	lib.Version = defaultVersion
	return lib
}

// DefaultLibraryName derives a default library name from an API path by stripping
// known prefixes (e.g., "google/cloud/", "google/api/") and returning all
// segments except the last one, joined by dashes.
func DefaultLibraryName(api string) string {
	path := api
	if idx := strings.LastIndex(api, "/"); idx != -1 {
		path = api[:idx]
	}
	for _, p := range knownPrefixes {
		if strings.HasPrefix(path, p) {
			path = strings.TrimPrefix(path, p)
			break
		}
	}
	return strings.ReplaceAll(path, "/", "-")
}
