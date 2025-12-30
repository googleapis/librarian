// Copyright 2025 Google LLC
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

package rust

import (
	"maps"
	"os"
	"path"
	"slices"
)

func FindCargoManifests(files []string) []string {
	isCandidate := func(parent string) bool {
		return parent != "/" && parent != "." && parent != ""
	}
	unique := map[string]bool{}
	for _, f := range files {
		if d := path.Dir(f); isCandidate(d) {
			unique[d] = true
		}
	}

	candidates := slices.Collect(maps.Keys(unique))
	manifests := map[string]bool{}
	for len(candidates) != 0 {
		d := candidates[len(candidates)-1]
		candidates = candidates[0 : len(candidates)-1]
		manifest := path.Join(d, "Cargo.toml")
		if _, ok := manifests[manifest]; ok {
			continue
		}
		if _, err := os.Stat(manifest); err == nil {
			manifests[manifest] = true
		} else if parent := path.Dir(d); isCandidate(parent) {
			candidates = append(candidates, parent)
		}
	}

	list := slices.Collect(maps.Keys(manifests))
	slices.Sort(list)
	return list
}
