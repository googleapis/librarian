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
	"bytes"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestUpdateCargoVersionSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	testhelper.AddCrate(t, tmpDir, "google-cloud-storage")
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := UpdateCargoVersion(manifest, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if idx := bytes.Index(contents, []byte(`version                = "1.2.3"`)); idx == -1 {
		t.Errorf("version 1.2.3 not found in %s", contents)
	}
}

func TestUpdateCargoVersionMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := UpdateCargoVersion(manifest, "1.2.3"); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestUpdateCargoVersionMissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := path.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(manifest, []byte("[package]\nname=\"foo\""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCargoVersion(manifest, "1.2.3"); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestUpdateManifestSuccess(t *testing.T) {
	const tag = "update-manifest-success"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)
	name := path.Join("src", "storage", "Cargo.toml")

	version, crateName, err := UpdateManifest("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("1.1.0", version); diff != "" {
		t.Errorf("version mismatch (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff("google-cloud-storage", crateName); diff != "" {
		t.Errorf("crate name mismatch (-want, +got):\n%s", diff)
	}
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.Index(contents, []byte(`version                = "1.1.0"`))
	if idx == -1 {
		t.Errorf("expected version = 1.1.0 in new file, got=%s", contents)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "update version", "."); err != nil {
		t.Fatal(err)
	}

	// Calling this a second time has no effect.
	version, crateName, err = UpdateManifest("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if version != "" || crateName != "" {
		t.Errorf("expected empty version and crate name, got %q and %q", version, crateName)
	}
	contents, err = os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	idx = bytes.Index(contents, []byte(`version                = "1.1.0"`))
	if idx == -1 {
		t.Errorf("expected version = 1.1.0 in new file, got=%s", contents)
	}
}

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

	needsBump, err := manifestVersionNeedsBump("git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if needsBump {
		t.Errorf("expected no need for a bump for %s", name)
	}
}
