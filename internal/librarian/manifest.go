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

package librarian

import (
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// loadReleasePleaseManifest reads and parses release-please manifest files matching pattern.
// If pattern is empty, it defaults to config.ReleasePleaseManifestPattern.
// It merges entries from all matching manifest files (such as bulk and individual manifests).
// Returns an empty map if no matching manifest files are found.
func loadReleasePleaseManifest(pattern string) (map[string]string, error) {
	if pattern == "" {
		pattern = config.ReleasePleaseManifestPattern
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		data, err := os.ReadFile(pattern)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return map[string]string{}, nil
			}
			return nil, err
		}
		var manifest map[string]string
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	}

	merged := make(map[string]string)
	for _, file := range matches {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var manifest map[string]string
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		maps.Copy(merged, manifest)
	}
	return merged, nil
}

// resolveVersion resolves the version for lib using the provided release-please manifest map.
// Resolution order:
// 1. Exact match by derived output directory (e.g., Node.js/Python "packages/google-cloud-memorystore").
// 2. Exact match by library name (e.g., Go standard packages "accessapproval").
// 3. Match Go major version module subpaths (e.g., "pubsub/v2").
// 4. Fallback to root "." for single-component repos.
// 5. Existing lib.Version set in librarian.yaml.
func resolveVersion(lib *config.Library, manifest map[string]string) string {
	if len(manifest) > 0 {
		if v, ok := manifest[lib.Output]; ok && v != "" {
			return v
		}
		if v, ok := manifest[lib.Name]; ok && v != "" {
			return v
		}
		if lib.Go != nil && lib.Go.ModulePathVersion != "" {
			if v, ok := manifest[lib.Name+"/"+lib.Go.ModulePathVersion]; ok && v != "" {
				return v
			}
		}
		if v, ok := manifest["."]; ok && v != "" {
			return v
		}
	}
	return lib.Version
}
