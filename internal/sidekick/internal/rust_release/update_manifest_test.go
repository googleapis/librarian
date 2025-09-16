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

package rustrelease

import (
	"bytes"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

func TestUpdateManifestSuccess(t *testing.T) {
	const tag = "update-manifest-success"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")

	got, err := updateManifest(&release, tag, name)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"google-cloud-storage"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.Index(contents, []byte("version = \"1.1.0\"\n"))
	if idx == -1 {
		t.Errorf("expected version = 1.1.0 in new file, got=%s", contents)
	}
	if err := external.Run("git", "commit", "-m", "update version", "."); err != nil {
		t.Fatal(err)
	}

	// Calling this a second time has no effect.
	got, err = updateManifest(&release, tag, name)
	if err != nil {
		t.Fatal(err)
	}
	want = nil
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestUpdateManifestBadDelta(t *testing.T) {
	const tag = "update-manifest-bad-delta"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")

	if got, err := updateManifest(&release, "invalid-tag", name); err == nil {
		t.Errorf("expected an error when using an invalid tag, got=%v", got)
	}
}

func TestUpdateManifestBadManifest(t *testing.T) {
	const tag = "update-manifest-bad-manifest"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}

	if got, err := updateManifest(&release, tag, name); err == nil {
		t.Errorf("expected an error when using an invalid tag, got=%v", got)
	}
}

func TestUpdateManifestBadContents(t *testing.T) {
	const tag = "update-manifest-bad-contents"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("invalid = {\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got, err := updateManifest(&release, tag, name); err == nil {
		t.Errorf("expected an error when using an invalid tag, got=%v", got)
	}
}

func TestUpdateManifestSkipUnpublished(t *testing.T) {
	const tag = "update-manifest-skip-unpublished"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, []byte("publish = false\n")...)
	if err := os.WriteFile(name, contents, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := updateManifest(&release, tag, name)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestUpdateManifestBadVersion(t *testing.T) {
	const tag = "update-manifest-bad-version"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	contents := `# Bad version
[package]
name = "google-cloud-storage"
version = "a.b.c"
`
	if err := os.WriteFile(name, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "introduce bad version", "."); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "tag", "bad-version-tag"); err != nil {
		t.Fatal(err)
	}

	if got, err := updateManifest(&release, "bad-version-tag", name); err == nil {
		t.Errorf("expected an error when using a bad version, got=%v", got)
	}
}

func TestUpdateManifestNoVersion(t *testing.T) {
	const tag = "update-manifest-no-version"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	contents := `# No Version
[package]
name = "google-cloud-storage"
`
	if err := os.WriteFile(name, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	if got, err := updateManifest(&release, tag, name); err == nil {
		t.Errorf("expected an error when using a bad version, got=%v", got)
	}
}

func TestUpdateManifestBadSidekickConfig(t *testing.T) {
	const tag = "update-manifest-bad-sidekick"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(path.Join("src", "storage", ".sidekick.toml"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	if got, err := updateManifest(&release, tag, name); err == nil {
		t.Errorf("expected an error when using a bad version, got=%v", got)
	}
}

func TestBumpPackageVersion(t *testing.T) {
	for _, test := range []struct {
		Input string
		Want  string
	}{
		{"1.2", "1.2"},
		{"1.2.3", "1.3.0"},
		{"1.2.3-alpha", "1.3.0-alpha"},
		{"0.1.2", "0.2.0"},
		{"0.1.2-alpha", "0.2.0-alpha"},
	} {
		got, err := bumpPackageVersion(test.Input)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.Want {
			t.Errorf("mismatch, want=%s, got=%s", test.Want, got)
		}
	}
}

func TestManifestVersionUpdatedSuccess(t *testing.T) {
	const tag = "manifest-version-update-success"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
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
	if err := external.Run("git", "commit", "-m", "updated version", "."); err != nil {
		t.Fatal(err)
	}

	updated, err := manifestVersionUpdated(&release, tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Errorf("expected a change for %s, got=%v", name, updated)
	}
}

func TestManifestVersionUpdatedNoChange(t *testing.T) {
	const tag = "manifest-version-update-no-change"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	updated, err := manifestVersionUpdated(&release, tag, name)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Errorf("expected no change for %s, got=%v", name, updated)
	}
}

func TestManifestVersionUpdatedBadDiff(t *testing.T) {
	const tag = "manifest-version-update-success"
	requireCommand(t, "git")
	setupForVersionBump(t, tag)
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	name := path.Join("src", "storage", "Cargo.toml")
	if updated, err := manifestVersionUpdated(&release, "not-a-valid-tag", name); err == nil {
		t.Errorf("expected an error with an valid tag, got=%v", updated)
	}
}
