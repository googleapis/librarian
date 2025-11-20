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
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

const (
	newLibRsContents = `pub fn hello() -> &'static str { "Hello World" }`
)

func TestMatchesBranchPointSuccess(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	remoteDir := setupForPublish(t, "v1.0.0")
	cloneRepository(t, remoteDir)
	if err := matchesBranchPoint(config); err != nil {
		t.Fatal(err)
	}
}

func TestMatchesBranchDiffError(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := setupForPublish(t, "v1.0.0")
	cloneRepository(t, remoteDir)
	if err := matchesBranchPoint(config); err == nil {
		t.Errorf("expected an error with an invalid branch")
	}
}

func TestMatchesDirtyCloneError(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := setupForPublish(t, "v1.0.0")
	cloneRepository(t, remoteDir)
	addCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run("git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}

	if err := matchesBranchPoint(config); err == nil {
		t.Errorf("expected an error with a dirty clone")
	}
}

func TestIsNewFile(t *testing.T) {
	const wantTag = "new-file-success"
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	setupForVersionBump(t, wantTag)
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(existingName, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	newName := path.Join("src", "storage", "src", "new.rs")
	if err := os.WriteFile(newName, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if isNewFile(&release, wantTag, existingName) {
		t.Errorf("file is not new but reported as such: %s", existingName)
	}
	if !isNewFile(&release, wantTag, newName) {
		t.Errorf("file is new but not reported as such: %s", newName)
	}
}

func TestIsNewFileDiffError(t *testing.T) {
	const wantTag = "new-file-success"
	release := config.Release{
		Remote:       "origin",
		Branch:       "main",
		Preinstalled: map[string]string{},
	}
	setupForVersionBump(t, wantTag)
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if isNewFile(&release, "invalid-tag", existingName) {
		t.Errorf("diff errors should return false for isNewFile(): %s", existingName)
	}
}

func TestFilesChangedSuccess(t *testing.T) {
	const wantTag = "release-2001-02-03"

	const echo = "/bin/echo"
	requireCommand(t, echo)
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	setupForVersionBump(t, wantTag)
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}

	got, err := filesChangedSince(&release, wantTag)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{name}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilesBadRef(t *testing.T) {
	const wantTag = "release-2002-03-04"

	const echo = "/bin/echo"
	requireCommand(t, echo)
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	setupForVersionBump(t, wantTag)
	if got, err := filesChangedSince(&release, "--invalid--"); err == nil {
		t.Errorf("expected an error with invalid tag, got=%v", got)
	}
}

func TestFilterNoFilter(t *testing.T) {
	input := []string{
		"src/storage/src/lib.rs",
		"src/storage/Cargo.toml",
		"src/storage/.repo-metadata.json",
		"src/generated/cloud/secretmanager/v1/.sidekick.toml",
		"src/generated/cloud/secretmanager/v1/Cargo.toml",
		"src/generated/cloud/secretmanager/v1/src/model.rs",
	}

	config := &config.Release{}
	got := filesFilter(config, input)
	want := input
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilterBasic(t *testing.T) {
	input := []string{
		"src/storage/src/lib.rs",
		"src/storage/Cargo.toml",
		"src/storage/.repo-metadata.json",
		"src/generated/cloud/secretmanager/v1/.sidekick.toml",
		"src/generated/cloud/secretmanager/v1/Cargo.toml",
		"src/generated/cloud/secretmanager/v1/src/model.rs",
	}

	config := &config.Release{
		IgnoredChanges: []string{
			".sidekick.toml",
			".repo-metadata.json",
		},
	}
	got := filesFilter(config, input)
	want := []string{
		"src/storage/src/lib.rs",
		"src/storage/Cargo.toml",
		"src/generated/cloud/secretmanager/v1/Cargo.toml",
		"src/generated/cloud/secretmanager/v1/src/model.rs",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilterSomeGlobs(t *testing.T) {
	input := []string{
		"doc/howto-1.md",
		"doc/howto-2.md",
	}

	config := &config.Release{
		IgnoredChanges: []string{
			".sidekick.toml",
			".repo-metadata.json",
			"doc/**",
		},
	}
	got := filesFilter(config, input)
	want := []string{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilterEmpty(t *testing.T) {
	input := []string{
		"",
	}

	config := &config.Release{
		IgnoredChanges: []string{
			".sidekick.toml",
			".repo-metadata.json",
			"doc/**",
		},
	}
	got := filesFilter(config, input)
	want := []string{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
