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
	"os"
	"path/filepath"
	"strings"
)

// PackageDirectory returns the root directory of the Swift package containing dir.
// It searches upward from dir for a Package.swift file. If none is found, it strips any
// /Sources/... suffix or returns dir. If dir is empty, it returns an empty string.
func PackageDirectory(dir string) string {
	if dir == "" {
		return ""
	}
	current := filepath.Clean(dir)
	for current != "." {
		if _, err := os.Stat(filepath.Join(current, "Package.swift")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	slashDir := filepath.ToSlash(filepath.Clean(dir))
	if before, _, ok := strings.Cut(slashDir, "/Sources/"); ok {
		return filepath.FromSlash(before)
	}
	return dir
}
