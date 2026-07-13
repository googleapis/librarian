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
	"os"

	"github.com/googleapis/librarian/internal/config"
)

// loadReleasePleaseManifest reads and parses .release-please-manifest.json
// if it exists at path. Returns an empty map if the file does not exist.
func loadReleasePleaseManifest(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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

// resolveVersion resolves the version for lib using the provided release-please manifest map.
// Resolution order:
// 1. Manifest key matching lib.Output (e.g., "packages/google-cloud-memorystore")
// 2. Manifest key matching lib.Name (e.g., "google-cloud-memorystore")
// 3. Manifest key matching "." (for single-component repos)
// 4. Existing lib.Version set in librarian.yaml.
func resolveVersion(lib *config.Library, manifest map[string]string) string {
	if len(manifest) > 0 {
		if v, ok := manifest[lib.Output]; ok && v != "" {
			return v
		}
		if v, ok := manifest[lib.Name]; ok && v != "" {
			return v
		}
		if v, ok := manifest["."]; ok && v != "" {
			return v
		}
	}
	return lib.Version
}
