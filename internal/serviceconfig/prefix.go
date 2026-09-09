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
	"strings"
)

// MatchPrefix finds the longest matching prefix for apiPath in prefixes.
// It strips any API version, checks the exact path, and walks up parent paths.
//
// Returns the mapped value, remainder path, and true if found.
func MatchPrefix(apiPath string, prefixes map[string]string) (value, remainder string, ok bool) {
	if apiPath == "" || len(prefixes) == 0 {
		return "", "", false
	}
	path := strings.Trim(apiPath, "/")
	if val, ok := prefixes[path]; ok {
		return val, "", true
	}
	if v := ExtractVersion(path); v != "" {
		path = strings.TrimSuffix(strings.TrimSuffix(path, v), "/")
	}
	for curr := path; curr != ""; {
		if val, ok := prefixes[curr]; ok {
			return val, strings.TrimPrefix(path[len(curr):], "/"), true
		}
		idx := strings.LastIndex(curr, "/")
		if idx == -1 {
			break
		}
		curr = curr[:idx]
	}
	return "", "", false
}
