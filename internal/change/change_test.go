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
	"context"
	"os"
	"path"
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
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
	const wantTag = "v1.2.3"
	t.Chdir(t.TempDir())
	testhelpers.SetupRepo(t, wantTag)
	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	got, err := GetLastTag(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantTag {
		t.Errorf("GetLastTag() = %q, want %q", got, wantTag)
	}
}

func TestIsNewFile(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
	const wantTag = "new-file-success"
	t.Chdir(t.TempDir())
	// Don't use SetupRepo because we need to create a commit before the tag.
	if err := command.Run(ctx, "git", "init"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	existingName := "README.md"
	if err := os.WriteFile(existingName, []byte("old file"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "commit", "-m", "add readme"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "tag", wantTag); err != nil {
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
	if err := command.Run(ctx, "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "commit", "-m", "feat: changed storage"); err != nil {
		t.Fatal(err)
	}
	if IsNewFile(ctx, gitExe, wantTag, existingName) {
		t.Errorf("file is not new but reported as such: %s", existingName)
	}
	if !IsNewFile(ctx, gitExe, wantTag, newName) {
		t.Errorf("file is new but not reported as such: %s", newName)
	}
}

func TestIsNewFileDiffError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
	const wantTag = "new-file-success"
	t.Chdir(t.TempDir())
	testhelpers.SetupRepo(t, wantTag)
	cfg := &config.Release{}
	gitExe := cfg.GetExecutablePath("git")
	existingName := "README.md"
	if IsNewFile(ctx, gitExe, "invalid-tag", existingName) {
		t.Errorf("diff errors should return false for isNewFile(): %s", existingName)
	}
}

func TestFilesChangedSuccess(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
	const wantTag = "release-2001-02-03"
	release := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	t.Chdir(t.TempDir())
	testhelpers.SetupRepo(t, wantTag)
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.MkdirAll(path.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "commit", "-m", "feat: changed storage"); err != nil {
		t.Fatal(err)
	}

	got, err := FilesChangedSince(ctx, wantTag, release)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{name}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestFilesBadRef(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
	const wantTag = "release-2002-03-04"
	release := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	t.Chdir(t.TempDir())
	testhelpers.SetupRepo(t, wantTag)
	if got, err := FilesChangedSince(ctx, "--invalid--", release); err == nil {
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
	got := filesFilter(cfg, input)
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
	got := filesFilter(cfg, input)
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
	got := filesFilter(cfg, input)
	want := []string{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestAssertGitStatusClean(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	ctx := context.Background()
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
				testhelpers.SetupRepo(t, "release-1.2.3")
			},
			wantErr: false,
		},
		{
			name: "dirty",
			setup: func(t *testing.T) {
				testhelpers.SetupRepo(t, "release-1.2.3")
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
			err := AssertGitStatusClean(ctx, cfg.GetExecutablePath("git"))
			if (err != nil) != test.wantErr {
				t.Errorf("AssertGitStatusClean() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
