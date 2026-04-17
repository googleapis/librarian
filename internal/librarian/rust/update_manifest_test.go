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
	"errors"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/googleapis/librarian/internal/semver"

	"github.com/googleapis/librarian/internal/testhelper"
)

func TestShouldBumpManifestVersionSuccess(t *testing.T) {
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
	testhelper.RunGit(t, "commit", "-m", "updated version", ".")

	needsBump, err := shouldBumpManifestVersion(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if needsBump {
		t.Errorf("expected no need for a bump for %s", name)
	}
}

func TestShouldBumpManifestVersionNewCrate(t *testing.T) {
	const tag = "manifest-version-update-new-crate"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)

	testhelper.AddCrate(t, path.Join("src", "new"), "google-cloud-new")
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "new crate", ".")
	name := path.Join("src", "new", "Cargo.toml")

	needsBump, err := shouldBumpManifestVersion(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if needsBump {
		t.Errorf("no changes for new crates")
	}
}

func TestShouldBumpManifestVersionNoChange(t *testing.T) {
	const tag = "manifest-version-update-no-change"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)
	name := path.Join("src", "storage", "Cargo.toml")
	needsBump, err := shouldBumpManifestVersion(t.Context(), "git", tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if !needsBump {
		t.Errorf("expected no change for %s", name)
	}
}

func TestShouldBumpManifestVersionBadDiff(t *testing.T) {
	const tag = "manifest-version-update-success"
	testhelper.RequireCommand(t, "git")
	testhelper.SetupForVersionBump(t, tag)
	name := path.Join("src", "storage", "Cargo.toml")
	if updated, err := shouldBumpManifestVersion(t.Context(), "git", "not-a-valid-tag", name); err == nil {
		t.Errorf("expected an error with an valid tag, got=%v", updated)
	}
}

func TestUpdateCargoVersion(t *testing.T) {
	content := "[package]\nname = \"test-crate\"\nversion = \"1.0.0\"\n"
	filePath := setupTestCargoFile(t, content)
	newVersion, err := semver.Parse("2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCargoVersion(filePath, newVersion); err != nil {
		t.Fatal(err)
	}
	updatedContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	expected := "[package]\nname = \"test-crate\"\nversion                = \"2.0.0\"\n"
	if diff := cmp.Diff(expected, string(updatedContent)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateCargoVersion_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name:    "no version field",
			content: "[package]\nname = \"test-crate\"\n",
			wantErr: ErrNoVersionField,
		},
		{
			name:    "file not found",
			content: "",
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := setupTestCargoFile(t, test.content)
			newVersion, _ := semver.Parse("2.0.0")
			err := updateCargoVersion(filePath, newVersion)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("updateCargoVersion() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestUpdateWorkspaceVersion(t *testing.T) {
	for _, test := range []struct {
		name      string
		content   string
		crateName string
		want      string
	}{
		{
			name:      "success",
			content:   "[workspace.dependencies]\ntest-crate = { version = \"1.0.0\", path = \"src/test-crate\" }\n",
			crateName: "test-crate",
			want:      "[workspace.dependencies]\ntest-crate = { version = \"2.0.0\", path = \"src/test-crate\" }\n",
		},
		{
			name:      "no-op",
			content:   "[workspace.dependencies]\nother-crate = { version = \"1.0.0\", path = \"src/other-crate\" }\n",
			crateName: "test-crate",
			want:      "[workspace.dependencies]\nother-crate = { version = \"1.0.0\", path = \"src/other-crate\" }\n",
		},
		{
			name:      "renamed dependency",
			content:   "[workspace.dependencies]\nwkt = { version = \"1.0.0\", package = \"google-cloud-wkt\" }\n",
			crateName: "google-cloud-wkt",
			want:      "[workspace.dependencies]\nwkt = { version = \"2.0.0\", package = \"google-cloud-wkt\" }\n",
		},
		{
			name:      "renamed dependency reverse order",
			content:   "[workspace.dependencies]\nwkt = { package = \"google-cloud-wkt\", version = \"1.0.0\" }\n",
			crateName: "google-cloud-wkt",
			want:      "[workspace.dependencies]\nwkt = { package = \"google-cloud-wkt\", version = \"2.0.0\" }\n",
		},
		{
			name:      "inline comment ignored",
			content:   "[workspace.dependencies]\nother-crate = { version = \"1.0.0\" } # package = \"google-cloud-wkt\"\n",
			crateName: "google-cloud-wkt",
			want:      "[workspace.dependencies]\nother-crate = { version = \"1.0.0\" } # package = \"google-cloud-wkt\"\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := setupTestCargoFile(t, test.content)
			newVersion, _ := semver.Parse("2.0.0")
			if err := updateWorkspaceVersion(filePath, test.crateName, newVersion); err != nil {
				t.Fatal(err)
			}
			updatedContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(updatedContent)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateWorkspaceVersion_Error(t *testing.T) {
	newVersion, _ := semver.Parse("2.0.0")
	err := updateWorkspaceVersion("non-existent-file", "test-crate", newVersion)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("updateWorkspaceVersion() error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func setupTestCargoFile(t *testing.T, content string) string {
	t.Helper()
	if content == "" {
		return "non-existent-file"
	}
	dir := t.TempDir()
	filePath := path.Join(dir, "Cargo.toml")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return filePath
}
