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

package git

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const (
	newLibRsContents = "pub fn hello() -> &'static str { \"Hello World\" }"
)

func TestGetLastTag(t *testing.T) {
	const wantTag = "v1.2.3"
	remoteDir := testhelper.SetupRepoWithChange(t, wantTag)
	testhelper.CloneRepository(t, remoteDir)
	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	got, err := GetLastTag(t.Context(), command.GetExecutablePath(cfg.Preinstalled, "git"), cfg.Remote, cfg.Branch)
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
	_, err := GetLastTag(t.Context(), command.GetExecutablePath(cfg.Preinstalled, "git"), cfg.Remote, cfg.Branch)
	if err == nil {
		t.Fatal("expected an error but got none")
	}
	if !strings.Contains(err.Error(), "fatal: not a git repository") && !strings.Contains(err.Error(), "exit status 128") {
		t.Errorf("expected git error, got: %v", err)
	}
}

func TestIsNewFileSuccess(t *testing.T) {
	testhelper.SetupForVersionBump(t, "dummy-tag")
	// Get the HEAD commit hash, which serves as a unique reference for this test.
	cmd := exec.CommandContext(t.Context(), "git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	headCommit := strings.TrimSpace(string(out))
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(existingName, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Release{}
	gitExe := command.GetExecutablePath(cfg.Preinstalled, "git")

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
	if IsNewFile(t.Context(), gitExe, headCommit, existingName) {
		t.Errorf("file is not new but reported as such: %s", existingName)
	}
	if !IsNewFile(t.Context(), gitExe, headCommit, newName) {
		t.Errorf("file is new but not reported as such: %s", newName)
	}
}

func TestIsNewFileDiffError(t *testing.T) {
	const wantTag = "new-file-success"
	t.Chdir(t.TempDir())
	testhelper.SetupForVersionBump(t, wantTag)
	cfg := &config.Release{}
	gitExe := command.GetExecutablePath(cfg.Preinstalled, "git")
	existingName := path.Join("src", "storage", "src", "lib.rs")
	if IsNewFile(t.Context(), gitExe, "invalid-tag", existingName) {
		t.Errorf("diff errors should return false for isNewFile(): %s", existingName)
	}
}

func TestFilesChangedSuccess(t *testing.T) {
	const wantTag = "release-2001-02-03"
	remoteDir := testhelper.SetupRepoWithChange(t, wantTag)
	testhelper.CloneRepository(t, remoteDir)

	got, err := FilesChangedSince(t.Context(), wantTag, command.GetExecutablePath(nil, "git"), nil)
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
	remoteDir := testhelper.SetupRepoWithChange(t, wantTag)
	testhelper.CloneRepository(t, remoteDir)
	if got, err := FilesChangedSince(t.Context(), "--invalid--", command.GetExecutablePath(nil, "git"), nil); err == nil {
		t.Errorf("expected an error with invalid tag, got=%v", got)
	}
}

func TestFilterNoFilter(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
		wantErr error
	}{
		{
			name: "clean",
			setup: func(t *testing.T) {
				remoteDir := testhelper.SetupRepoWithChange(t, "release-1.2.3")
				testhelper.CloneRepository(t, remoteDir)
			},
			wantErr: nil,
		},
		{
			name: "dirty",
			setup: func(t *testing.T) {
				remoteDir := testhelper.SetupRepoWithChange(t, "release-1.2.3")
				testhelper.CloneRepository(t, remoteDir)
				if err := os.WriteFile("dirty.txt", []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: ErrGitStatusUnclean,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			test.setup(t)
			err := AssertGitStatusClean(t.Context(), command.GetExecutablePath(cfg.Preinstalled, "git"))
			if !errors.Is(err, test.wantErr) {
				t.Errorf("AssertGitStatusClean() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestMatchesBranchPointSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "v1.0.0")
	testhelper.CloneRepository(t, remoteDir)
	if err := MatchesBranchPoint(t.Context(), "git", config.Remote, config.Branch); err != nil {
		t.Fatal(err)
	}
}

func TestMatchesBranchDiffError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "v1.0.0")
	testhelper.CloneRepository(t, remoteDir)
	if err := MatchesBranchPoint(t.Context(), "git", config.Remote, config.Branch); err == nil {
		t.Errorf("expected an error with an invalid branch")
	}
}

func TestMatchesDirtyCloneError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "not-a-valid-branch",
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "v1.0.0")
	testhelper.CloneRepository(t, remoteDir)
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}

	if err := MatchesBranchPoint(t.Context(), "git", config.Remote, config.Branch); err == nil {
		t.Errorf("expected an error with a dirty clone")
	}
}

func TestShowFileAtRemoteBranch(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	remoteDir := testhelper.SetupRepo(t)
	testhelper.CloneRepository(t, remoteDir)
	got, err := ShowFileAtRemoteBranch(t.Context(), "git", "origin", "main", testhelper.ReadmeFile)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(testhelper.ReadmeContents, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestShowFileAtRemoteBranch_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	remoteDir := testhelper.SetupRepo(t)
	testhelper.CloneRepository(t, remoteDir)
	_, err := ShowFileAtRemoteBranch(t.Context(), "git", "origin", "main", "does_not_exist")
	if err == nil {
		t.Fatal("expected an error showing file that should not exist")
	}
	if !errors.Is(err, errGitShow) {
		t.Errorf("expected errGitShow but got %v", err)
	}
}

func TestShowFileAtRevision(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	opts := testhelper.SetupOptions{
		WithChanges: []string{testhelper.ReadmeFile},
	}
	testhelper.Setup(t, opts)
	// The commit before HEAD should have the known sample content.
	gotOriginal, err := ShowFileAtRevision(t.Context(), "git", "HEAD~", testhelper.ReadmeFile)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(testhelper.ReadmeContents, gotOriginal); diff != "" {
		t.Errorf("mismatch for original file content (-want, +got):\n%s", diff)
	}
	// The current commit should be what we currently have on disk.
	gotModified, err := ShowFileAtRevision(t.Context(), "git", "HEAD", testhelper.ReadmeFile)
	if err != nil {
		t.Fatal(err)
	}
	contentOnDisk, err := os.ReadFile(testhelper.ReadmeFile)
	if err != nil {
		t.Fatal(err)
	}
	wantModified := strings.TrimSuffix(string(contentOnDisk), "\n")
	if diff := cmp.Diff(wantModified, gotModified); diff != "" {
		t.Errorf("mismatch for modified file content (-want, +got):\n%s", diff)
	}
}

func TestShowFileAtRevision_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.SetupRepo(t)
	_, err := ShowFileAtRevision(t.Context(), "git", "HEAD", "does_not_exist")
	if err == nil {
		t.Fatal("expected an error showing file that should not exist")
	}
	if !errors.Is(err, errGitShow) {
		t.Errorf("expected errGitShow but got %v", err)
	}
}

func TestCheckVersion(t *testing.T) {
	t.Parallel()
	testhelper.RequireCommand(t, "git")

	if err := CheckVersion(t.Context(), "git"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckVersion_Error(t *testing.T) {
	t.Parallel()
	if err := CheckVersion(t.Context(), "command_that_does_not_exist"); err == nil {
		t.Errorf("expected an error checking git version execution, but did not get one")
	}
}

func TestCheckRemoteURL(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	remoteDir := testhelper.SetupRepo(t)
	testhelper.CloneRepository(t, remoteDir)

	if err := CheckRemoteURL(t.Context(), "git", testhelper.TestRemote); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRemoteURL_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	remoteDir := testhelper.SetupRepo(t)
	testhelper.CloneRepository(t, remoteDir)

	if err := CheckRemoteURL(t.Context(), "git", "remote_that_does_not_exist"); err == nil {
		t.Errorf("expected an error checking for a remote URL, but did not get one")
	}
}

func TestFindCommitsForPath(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	opts := testhelper.SetupOptions{
		WithChanges: []string{testhelper.ReadmeFile},
	}
	testhelper.Setup(t, opts)
	for _, test := range []struct {
		name       string
		path       string
		wantLength int
	}{
		{
			name:       "README file with changes",
			path:       testhelper.ReadmeFile,
			wantLength: 2,
		},
		{
			name:       "non-existent path",
			path:       "this/path/does/not/exist",
			wantLength: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := FindCommitsForPath(t.Context(), "git", test.path)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantLength != len(got) {
				t.Errorf("want %d changes, got %d", test.wantLength, len(got))
			}
			sampleHash := "bbeebf51301cfb45612db9869ec6dd8fa067d3fc"
			for _, hash := range got {
				if len(hash) != len(sampleHash) {
					t.Errorf("expected each commit hash to have length %d; got hash %s", len(sampleHash), hash)
				}
			}
		})
	}
}

func TestFindCommitsForPath_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.SetupRepo(t)
	// It's invalid to try to get the log for a path outside the repo
	if _, err := FindCommitsForPath(t.Context(), "git", ".."); err == nil {
		t.Errorf("expected an error finding commits for path outside the repo, but did not get one")
	}
}
