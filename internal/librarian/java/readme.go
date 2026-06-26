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
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Matches lowercase/digit followed by uppercase (e.g., "FooBar" -> "Foo Bar").
	camelCaseRegexp = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// decamelize converts CamelCase string to space-separated string (e.g. "CamelCase" -> "Camel Case").
func decamelize(value string) string {
	return strings.TrimSpace(camelCaseRegexp.ReplaceAllString(value, `$1 $2`))
}

// isProductionSample reports whether the given entry represents a production Java source file
// located under a standard "/src/main/java/" path.
func isProductionSample(d os.DirEntry, path string) bool {
	return !d.IsDir() &&
		strings.HasSuffix(path, ".java") &&
		strings.Contains(filepath.ToSlash(path), "/src/main/java/")
}
