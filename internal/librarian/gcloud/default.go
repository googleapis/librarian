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

package gcloud

import (
	"path/filepath"
	"strings"
)

// DeriveAPIPath returns the canonical API path for a gcloud library name. For
// example: accessapproval -> google/cloud/accessapproval/v1.
func DeriveAPIPath(name string) string {
	// TODO(https://github.com/googleapis/librarian/issues/5862): use the
	// actual API version from the service configuration or the library's
	// metadata rather than hardcoding "v1". At the moment we override this
	// with apis.path in librarian.yaml.
	return "google/cloud/" + name + "/v1"
}

// DefaultLibraryName returns the gcloud library name for an API path. For
// example: google/cloud/accessapproval/v1 -> accessapproval.
//
// The library name is the segment after the leading google/cloud/ (or
// google/) prefix and before the trailing version. APIs that do not match
// the expected shape fall through to a "/"-replaced fallback.
func DefaultLibraryName(api string) string {
	trimmed := strings.TrimPrefix(api, "google/cloud/")
	if trimmed == api {
		trimmed = strings.TrimPrefix(api, "google/")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		return strings.ReplaceAll(api, "/", "-")
	}
	return parts[0]
}

// DefaultOutput returns the output directory for a gcloud library. The
// directory is a subdirectory of defaultOutput named after the library. When
// defaultOutput is empty, "generated" is used so that a fresh librarian.yaml
// without an explicit default.output still produces sensible paths.
func DefaultOutput(name, defaultOutput string) string {
	if defaultOutput == "" {
		defaultOutput = "generated"
	}
	return filepath.Join(defaultOutput, name)
}
