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

package change

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

const (
	newLibRsContents = "pub fn hello() -> &'static str { \"Hello World\" }"
)

func TestGetLastTag(t *testing.T) {
	const wantTag = "v1.2.3"
	remoteDir := testhelpers.SetupForPublish(t, wantTag)
	testhelpers.CloneRepository(t, remoteDir)
	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	got, err := GetLastTag(t.Context(), cfg.GetExecutablePath("git"), cfg.Remote, cfg.Branch)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantTag {
		t.Errorf("GetLastTag() = %q, want %q", got, wantTag)
	}
}

func TestLastTagGitError(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	_, err := GetLastTag(t.Context(), cfg.GetExecutablePath("git"), cfg.Remote, cfg.Branch)
	if err == nil {
		t.Fatal("expected an error but got none")
	}
	if !strings.Contains(err.Error(), "fatal: not a git repository") && !strings.Contains(err.Error(), "exit status 128") {
		t.Errorf("expected git error, got: %v", err)
	}
}

func TestIsNewFile(t *testing.T) {
	const wantTag = "new-file-success"
	testhelpers.SetupForVersionBump(t, wantTag)
	if err := command.Run(t.Context(), "git", "tag", wantTag); err != nil {
		t.Fatal(err)
	}
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(existingName, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Release{}
	gitExe := cfg.GetExecutablePath("git")

	newName := path.Join("src", "storage", "src", "new.rs")
	if err := os.MkdirAll(path.Dir(newName), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newName, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if IsNewFile(t.Context(), gitExe, wantTag, existingName) {
		t.Errorf("file is not new but reported as such: %s", existingName)
	}
	if !IsNewFile(t.Context(), gitExe, wantTag, newName) {
		t.Errorf("file is new but not reported as such: %s", newName)
	}
}

func TestIsNewFileDiffError(t *testing.T) {
	const wantTag = "new-file-success"
	t.Chdir(t.TempDir())
	testhelpers.SetupForVersionBump(t, wantTag)
	cfg := &config.Release{}
	gitExe := cfg.GetExecutablePath("git")
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if IsNewFile(t.Context(), gitExe, "invalid-tag", existingName) {
		t.Errorf("diff errors should return false for isNewFile(): %s", existingName)
	}
}

func TestFilesChangedSuccess(t *testing.T) {
	const wantTag = "release-2001-02-03"
	release := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	remoteDir := testhelpers.SetupForPublish(t, wantTag)
	testhelpers.CloneRepository(t, remoteDir)

	got, err := FilesChangedSince(t.Context(), wantTag, release.GetExecutablePath("git"), release.IgnoredChanges)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{path.Join("src", "storage", "src", "lib.rs")}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilesBadRef(t *testing.T) {
	const wantTag = "release-2002-03-04"
	release := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	remoteDir := testhelpers.SetupForPublish(t, wantTag)
	testhelpers.CloneRepository(t, remoteDir)
	if got, err := FilesChangedSince(t.Context(), "--invalid--", release.GetExecutablePath("git"), release.IgnoredChanges); err == nil {
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

	cfg := &config.Release{}
	got := filesFilter(cfg.IgnoredChanges, input)
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

	cfg := &config.Release{
		IgnoredChanges: []string{
			".sidekick.toml",
			".repo-metadata.json",
		},
	}
	got := filesFilter(cfg.IgnoredChanges, input)
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

	cfg := &config.Release{
		IgnoredChanges: []string{
			".sidekick.toml",
			".repo-metadata.json",
			"doc/**",
		},
	}
	got := filesFilter(cfg.IgnoredChanges, input)
	want := []string{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestAssertGitStatusClean(t *testing.T) {
	cfg := &config.Release{
		Preinstalled: map[string]string{
			"git": "git",
		},
	}
	for _, test := range []struct {
		name    string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "clean",
			setup: func(t *testing.T) {
				remoteDir := testhelpers.SetupForPublish(t, "release-1.2.3")
				testhelpers.CloneRepository(t, remoteDir)
			},
			wantErr: false,
		},
		{
			name: "dirty",
			setup: func(t *testing.T) {
				remoteDir := testhelpers.SetupForPublish(t, "release-1.2.3")
				testhelpers.CloneRepository(t, remoteDir)
				if err := os.WriteFile("dirty.txt", []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			test.setup(t)
			err := AssertGitStatusClean(t.Context(), cfg.GetExecutablePath("git"))
			if (err != nil) != test.wantErr {
				t.Errorf("AssertGitStatusClean() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
