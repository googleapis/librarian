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

package serviceconfig

import (
	"regexp"
	"strings"
)

var versionRegex = regexp.MustCompile(`^v\d+(?:(alpha|beta)\d*)?$`)

// ExtractVersion extracts the version from the given API path.
// It searches for the last path component that matches the version pattern (e.g., v1, v1beta1).
func ExtractVersion(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if IsVersion(parts[i]) {
			return parts[i]
		}
	}
	return ""
}

// IsVersion returns true if the given string is a valid API version.
func IsVersion(s string) bool {
	return versionRegex.MatchString(s)
}
