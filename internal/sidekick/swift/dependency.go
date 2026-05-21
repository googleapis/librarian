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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Dependency wraps config.SwiftDependency to add helper methods.
type Dependency struct {
	config.SwiftDependency
}

// LocalName returns the name of the dependency when used in a `Package.swift` file.
//
// For local dependencies this is the last directory in the path. For external dependencies this is
// the last directory of the URL path.
func (dep *Dependency) LocalName() string {
	var source string
	if dep.Path != "" {
		source = dep.Path
	} else {
		source = strings.TrimSuffix(dep.URL, ".git")
	}
	source = strings.Trim(source, "/")
	idx := strings.LastIndex(source, "/")
	if idx == -1 {
		return source
	}
	return source[idx+1:]
}
