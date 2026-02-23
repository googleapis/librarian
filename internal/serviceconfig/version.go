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
// It only checks the last path component (the leaf directory).
func ExtractVersion(path string) string {
	v := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		v = path[i+1:]
	}
	if IsVersion(v) {
		return v
	}
	return ""
}

// IsVersion returns true if the given string is a valid API version.
func IsVersion(s string) bool {
	return versionRegex.MatchString(s)
}
