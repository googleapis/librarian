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
				remoteDir := testhelper.SetupRepoWithChange(t, "release-1.2.3")
				testhelper.CloneRepository(t, remoteDir)
			},
			wantErr: false,
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
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			test.setup(t)
			err := AssertGitStatusClean(t.Context(), command.GetExecutablePath(cfg.Preinstalled, "git"))
			if (err != nil) != test.wantErr {
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

func TestChangesInDirectorySinceTag(t *testing.T) {
	for _, test := range []struct {
		name string
		dir  string
		want int
	}{
		{
			name: "changes exist in directory",
			dir:  "src/storage",
			want: 1,
		},
		{
			name: "changes do not exist in directory",
			dir:  "src/gax-internal",
			want: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testhelper.RequireCommand(t, "git")
			tag := "v1.2.3"
			remoteDir := testhelper.SetupRepoWithChange(t, tag)
			testhelper.CloneRepository(t, remoteDir)
			got, err := ChangesInDirectorySinceTag(t.Context(), "git", tag, test.dir)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("ChangesInDirectorySinceTag() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestGetFileChangesSinceLastTag(t *testing.T) {
	for _, test := range []struct {
		name                string
		setup               func(t *testing.T)
		ignoredChanges      []string
		wantFiles           []string
		wantTag             string
		remote              string
		branch              string
		preinstalled        map[string]string
		wantErr             bool
		assertErr           func(t *testing.T, err error)
		assertFileExistence bool
	}{
		{
			name: "clean repository",
			setup: func(t *testing.T) {
				remoteDir := setupBareRepoWithChange(t, "v1.2.3")
				testhelper.CloneRepository(t, remoteDir)
			},
			remote:    "origin",
			branch:    "main",
			wantTag:   "v1.2.3",
			wantFiles: []string{path.Join("src", "storage", "src", "lib.rs")},
		},
		{
			name: "dirty repository",
			setup: func(t *testing.T) {
				remoteDir := testhelper.SetupRepoWithChange(t, "v1.2.3")
				testhelper.CloneRepository(t, remoteDir)
				if err := os.WriteFile("dirty.txt", []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			remote:  "origin",
			branch:  "main",
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "git working directory is not clean") {
					t.Errorf("GetFileChangesSinceLastTag() error = %v, wantErr containing 'git working directory is not clean'", err)
				}
			},
		},
		{
			name: "unpushed changes",
			setup: func(t *testing.T) {
				remoteDir := testhelper.SetupRepoWithChange(t, "v1.2.3")
				testhelper.CloneRepository(t, remoteDir)
				testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
				if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
					t.Fatal(err)
				}
				if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub"); err != nil {
					t.Fatal(err)
				}
			},
			remote:  "origin",
			branch:  "main",
			wantErr: true,
			assertErr: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "the local repository does not match its branch point") {
					t.Errorf("GetFileChangesSinceLastTag() error = %v, wantErr containing 'the local repository does not match its branch point'", err)
				}
			},
		},
		{
			name: "ignored changes",
			setup: func(t *testing.T) {
				remoteDir := setupBareRepoWithChange(t, "v1.2.3")
				testhelper.CloneRepository(t, remoteDir)
				if err := os.WriteFile(".repo-metadata.json", []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := command.Run(t.Context(), "git", "add", ".repo-metadata.json"); err != nil {
					t.Fatal(err)
				}
				if err := command.Run(t.Context(), "git", "commit", "-m", "feat: added repo-metadata.json"); err != nil {
					t.Fatal(err)
				}
				if err := command.Run(t.Context(), "git", "push", "origin", "main"); err != nil {
					t.Fatal(err)
				}
			},
			remote:         "origin",
			branch:         "main",
			ignoredChanges: []string{".repo-metadata.json"},
			wantTag:        "v1.2.3",
			wantFiles:      []string{path.Join("src", "storage", "src", "lib.rs")},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Helper()
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			test.setup(t)
			files, lastTag, err := GetFileChangesSinceLastTag(t.Context(), test.preinstalled, test.remote, test.branch, test.ignoredChanges)
			if (err != nil) != test.wantErr {
				t.Errorf("GetFileChangesSinceLastTag() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if test.assertErr != nil {
				test.assertErr(t, err)
			}
			if lastTag != test.wantTag {
				t.Errorf("GetFileChangesSinceLastTag() lastTag = %q, want %q", lastTag, test.wantTag)
			}
			if diff := cmp.Diff(test.wantFiles, files); diff != "" {
				t.Errorf("GetFileChangesSinceLastTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func setupBareRepoWithChange(t *testing.T, wantTag string) string {
	// Create a bare repository to act as the remote.
	bareRepoDir := t.TempDir()
	if err := command.Run(t.Context(), "git", "init", "--bare", bareRepoDir); err != nil {
		t.Fatal(err)
	}
	testhelper.SetupRepoWithChange(t, wantTag)
	if err := command.Run(t.Context(), "git", "remote", "add", "origin", bareRepoDir); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "push", "-u", "origin", "main", "--tags"); err != nil {
		t.Fatal(err)
	}
	return bareRepoDir
}
