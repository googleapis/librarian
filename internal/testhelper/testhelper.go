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

// Package testhelper provides helper functions for tests.
// These are used across packages
package testhelper

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googleapis/librarian/internal/command"
)

// RequireCommand skips the test if the specified command is not found in PATH.
// Use this to skip tests that depend on external tools like protoc, cargo, or
// taplo, so that `go test ./...` will always pass on a fresh clone of the
// repo.
func RequireCommand(t *testing.T, cmd string) {
	t.Helper()
	if _, err := exec.LookPath(cmd); err != nil {
		t.Skipf("skipping test because %s is not installed", cmd)
	}
}

const (
	// InitialCargoContents defines the initial content for a Cargo.toml file.
	InitialCargoContents = `# Example Cargo file
[package]
name    = "%s"
version = "1.0.0"
`

	// InitialLibRsContents defines the initial content for a lib.rs file.
	initialLibRsContents = `pub fn test() -> &'static str { "Hello World" }`

	// NewLibRsContents defines new content for a lib.rs file for testing changes.
	NewLibRsContents = `pub fn hello() -> &'static str { "Hello World" }`

	// ReadmeFile is the local file path for the README.md file initialized in
	// the test repo.
	ReadmeFile = "README.md"

	// ReadmeContents is the contents of the [ReadmeFile] initialized in the
	// test repo.
	ReadmeContents = "# Empty Repo"

	// TestRemote is the name of a remote source for the test repository.
	TestRemote = "test"

	// testRemoteURL is the URL set for the [TestRemote] in the test repository.
	testRemoteURL = "https://example.com/git.git"
)

// SetupForVersionBump sets up a git repository for testing version bumping scenarios.
func SetupForVersionBump(t *testing.T, wantTag string) {
	remoteDir := t.TempDir()
	ContinueInNewGitRepository(t, remoteDir)
	initRepositoryContents(t)
	if err := command.Run(t.Context(), "git", "tag", wantTag); err != nil {
		t.Fatal(err)
	}
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	if err := command.Run(t.Context(), "git", "clone", remoteDir, "."); err != nil {
		t.Fatal(err)
	}
	configNewGitRepository(t)
}

// ContinueInNewGitRepository initializes a new git repository in a temporary directory
// and changes the current working directory to it.
func ContinueInNewGitRepository(t *testing.T, tmpDir string) {
	t.Helper()
	RequireCommand(t, "git")
	t.Chdir(tmpDir)
	if err := command.Run(t.Context(), "git", "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	configNewGitRepository(t)
}

func configNewGitRepository(t *testing.T) {
	if err := command.Run(t.Context(), "git", "config", "user.email", "test@test-only.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "config", "user.name", "Test Account"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "remote", "add", TestRemote, testRemoteURL); err != nil {
		t.Fatal(err)
	}
}

func initRepositoryContents(t *testing.T) {
	t.Helper()
	RequireCommand(t, "git")
	if err := os.WriteFile(ReadmeFile, []byte(ReadmeContents), 0644); err != nil {
		t.Fatal(err)
	}
	AddCrate(t, path.Join("src", "storage"), "google-cloud-storage")
	AddCrate(t, path.Join("src", "gax-internal"), "google-cloud-gax-internal")
	AddCrate(t, path.Join("src", "gax-internal", "echo-server"), "echo-server")
	addGeneratedCrate(t, path.Join("src", "generated", "cloud", "secretmanager", "v1"), "google-cloud-secretmanager-v1")
	if err := command.Run(t.Context(), "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "initial version"); err != nil {
		t.Fatal(err)
	}
}

func addGeneratedCrate(t *testing.T, location, name string) {
	t.Helper()
	AddCrate(t, location, name)
	if err := os.WriteFile(path.Join(location, ".sidekick.toml"), []byte("# initial version"), 0644); err != nil {
		t.Fatal(err)
	}
}

// AddCrate creates a new Rust crate at the specified location with the given name.
func AddCrate(t *testing.T, location, name string) {
	t.Helper()
	_ = os.MkdirAll(path.Join(location, "src"), 0755)
	contents := []byte(fmt.Sprintf(InitialCargoContents, name))
	if err := os.WriteFile(path.Join(location, "Cargo.toml"), contents, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(location, "src", "lib.rs"), []byte(initialLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(location, ".repo-metadata.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
}

// SetupRepo creates a git repository for testing with some initial content. It
// returns the path of the remote repository.
func SetupRepo(t *testing.T) string {
	remoteDir := t.TempDir()
	ContinueInNewGitRepository(t, remoteDir)
	initRepositoryContents(t)
	return remoteDir
}

// SetupRepoWithChange creates a git repository for testing publish scenarios,
// including initial content, a tag, and a committed change.
// It returns the path to the remote repository.
func SetupRepoWithChange(t *testing.T, wantTag string) string {
	remoteDir := SetupRepo(t)
	if err := command.Run(t.Context(), "git", "tag", wantTag); err != nil {
		t.Fatal(err)
	}
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	return remoteDir
}

// CloneRepository clones the remote repository into a new temporary directory
// and changes the current working directory to the cloned repository.
func CloneRepository(t *testing.T, remoteDir string) {
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	if err := command.Run(t.Context(), "git", "clone", remoteDir, "."); err != nil {
		t.Fatal(err)
	}
	configNewGitRepository(t)
}
