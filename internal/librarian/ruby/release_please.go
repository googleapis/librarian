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

package ruby

import (
	"fmt"
	"strings"
)

// AddManifest inserts a new package and its filler entry into a Release Please manifest map.
func AddManifest(manifest map[string]string, name, version string) map[string]string {
	if _, ok := manifest[name]; ok {
		return manifest
	}
	manifest[name] = version
	manifest[name+"+FILLER"] = "0.0.0"
	return manifest
}

// AddPackage inserts a new package into a Release Please packages map.
func AddPackage(packages map[string]any, name string) map[string]any {
	if _, ok := packages[name]; ok {
		return packages
	}
	packages[name] = map[string]any{
		"component":    name,
		"version_file": toVersionFile(name),
	}
	return packages
}

func toVersionFile(name string) string {
	return fmt.Sprintf("lib/%s/version.rb", strings.ReplaceAll(name, "-", "/"))
}
