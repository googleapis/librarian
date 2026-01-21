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
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestManifestVersionNeedsBumpSuccess(t *testing.T) {
	const tag = "manifest-version-update-success"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)

	name := path.Join("src", "storage", "Cargo.toml")
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(contents), "\n")
	idx := slices.IndexFunc(lines, func(a string) bool { return strings.HasPrefix(a, "version ") })
	if idx == -1 {
		t.Fatalf("expected a line starting with `version ` in %v", lines)
	}
	lines[idx] = `version = "2.3.4"`
	if err := os.WriteFile(name, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "updated version", "."); err != nil {
		t.Fatal(err)
	}

	needsBump, err := ManifestVersionNeedsBump("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if needsBump {
		t.Errorf("expected no need for a bump for %s", name)
	}
}

func TestManifestVersionNeedsBumpNewCrate(t *testing.T) {
	const tag = "manifest-version-update-new-crate"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)

	testhelper.AddCrate(t, path.Join("src", "new"), "google-cloud-new")
	if err := command.Run(t.Context(), "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "new crate", "."); err != nil {
		t.Fatal(err)
	}
	name := path.Join("src", "new", "Cargo.toml")

	needsBump, err := ManifestVersionNeedsBump("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if needsBump {
		t.Errorf("no changes for new crates")
	}
}

func TestManifestVersionNeedsBumpNoChange(t *testing.T) {
	const tag = "manifest-version-update-no-change"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)
	name := path.Join("src", "storage", "Cargo.toml")
	needsBump, err := ManifestVersionNeedsBump("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if !needsBump {
		t.Errorf("expected no change for %s", name)
	}
}

func TestManifestVersionNeedsBumpBadDiff(t *testing.T) {
	const tag = "manifest-version-update-success"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)
	name := path.Join("src", "storage", "Cargo.toml")
	if updated, err := ManifestVersionNeedsBump("git", "not-a-valid-tag", name); err == nil {
		t.Errorf("expected an error with an valid tag, got=%v", updated)
	}
}
