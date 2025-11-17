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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	toml "github.com/pelletier/go-toml/v2"
)

func TestReleaseTomlSpec(t *testing.T) {
	// Verify the toml annotations for `Release` work as expected.
	input := `
[release]
remote = 'upstream'
branch = 'main'
ignored-changes = [
	".sidekick.toml",
	".repo-metadata.json",
	"**/examples/**",
]

[[release.tools.cargo]]
name = 'release-plz'
version = '1.2.3'

[[release.tools.cargo]]
name = 'workspaces'
version = '2.3.4'

[release.pre-installed]
cargo = '/bin/true'
git = '/bin/false'
`

	got := Config{}
	if err := toml.Unmarshal([]byte(input), &got); err != nil {
		t.Fatal(err)
	}

	want := Config{
		Release: &Release{
			Remote: "upstream",
			Branch: "main",
			Tools: map[string][]Tool{
				"cargo": {
					{Name: "release-plz", Version: "1.2.3"},
					{Name: "workspaces", Version: "2.3.4"},
				},
			},
			Preinstalled: map[string]string{
				"cargo": "/bin/true",
				"git":   "/bin/false",
			},
			IgnoredChanges: []string{
				".sidekick.toml",
				".repo-metadata.json",
				"**/examples/**",
			},
		},
	}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
