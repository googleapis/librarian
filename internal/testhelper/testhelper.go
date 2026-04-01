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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/yaml"
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
	RunGit(t, "tag", wantTag)
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	RunGit(t, "clone", remoteDir, ".")
	RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	configNewGitRepository(t)
}

// ContinueInNewGitRepository initializes a new git repository in a temporary directory
// and changes the current working directory to it.
func ContinueInNewGitRepository(t *testing.T, tmpDir string) {
	t.Helper()
	RequireCommand(t, "git")
	t.Chdir(tmpDir)
	RunGit(t, "init", "-b", config.BranchMain)
	configNewGitRepository(t)
}

func configNewGitRepository(t *testing.T) {
	RunGit(t, "config", "user.email", "test@test-only.com")
	RunGit(t, "config", "user.name", "Test Account")
	RunGit(t, "remote", "add", TestRemote, testRemoteURL)
}

func initRepositoryContents(t *testing.T) {
	t.Helper()
	RequireCommand(t, "git")
	if err := os.WriteFile(ReadmeFile, []byte(ReadmeContents), 0644); err != nil {
		t.Fatal(err)
	}
	AddCrate(t, sample.Lib1Output, sample.Lib1Name)
	AddCrate(t, sample.Lib2Output, sample.Lib2Name)
	AddCrate(t, path.Join(sample.Lib2Output, "echo-server"), "echo-server")
	addGeneratedCrate(t, path.Join("src", "generated", "cloud", "secretmanager", "v1"), "google-cloud-secretmanager-v1")
	RunGit(t, "add", ".")
	RunGit(t, "commit", "-m", "initial version")
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

// SetupOptions include the various options for configuring test setup.
type SetupOptions struct {
	// Clone indicates whether to clone the repository after setup is
	// complete. The clone uses [config.BranchMain].
	Clone bool
	// Config is the [config.Config] to write to librarian.yaml in the root
	// of the repo created.
	Config *config.Config
	// Dirty indicates if the cloned repository should be left in a dirty state,
	// with uncommitted files. Primarily used for error testing.
	Dirty bool
	// remoteDir is the directory of the repo created by [SetupRepo] that
	// is cloned when [Clone] is true. Internal only.
	remoteDir string
	// Tags is the list of tags that will be applied once all initial file set up is
	// complete.
	Tags []string
	// WithChanges is a list of file paths that should show as changed and be
	// committed after Tag has been applied.
	WithChanges []string
}

// Setup is a configurable test setup function that starts by creating a
// fresh test repository via [SetupRepo], to which it then applies the
// configured [SetupOptions].
func Setup(t *testing.T, opts SetupOptions) {
	t.Helper()
	dir := SetupRepo(t)
	opts.remoteDir = dir
	setup(t, opts)
}

func setup(t *testing.T, opts SetupOptions) {
	if opts.Config != nil {
		addLibrarianConfig(t, opts.Config)
	}
	for _, tag := range opts.Tags {
		RunGit(t, "tag", tag)
	}
	// Must be handled after tagging for tests that need to detect untagged
	// changes needing release.
	if len(opts.WithChanges) > 0 {
		for _, srcPath := range opts.WithChanges {
			touchFile(t, srcPath)
		}
		RunGit(t, "commit", "-m", "feat: changed file(s)", ".")
	}
	if opts.Clone {
		CloneRepository(t, opts.remoteDir)
	}
	if opts.Dirty {
		RunGit(t, "reset", "HEAD~1")
	}
}

func touchFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Append a new line to the end of each file to show as "changed".
	if _, err := fmt.Fprintln(f, ""); err != nil {
		t.Fatal(err)
	}
}

// addLibrarianConfig writes the provided librarian.yaml config to disk and
// commits it. Must be called after a Setup or a Clone.
func addLibrarianConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	if cfg == nil {
		return
	}
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	RunGit(t, "add", ".")
	RunGit(t, "commit", "-m", "chore: add/update librarian yaml", ".")
}

// SetupRepoWithChange creates a git repository for testing publish scenarios,
// including initial content, a tag, and a committed change.
// It returns the path to the remote repository.
func SetupRepoWithChange(t *testing.T, wantTag string) string {
	remoteDir := SetupRepo(t)
	RunGit(t, "tag", wantTag)
	name := path.Join(sample.Lib1Output, "src", "lib.rs")
	if err := os.WriteFile(name, []byte(NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	RunGit(t, "commit", "-m", "feat: changed storage", ".")
	return remoteDir
}

// CloneRepository clones the remote repository into a new temporary directory
// and changes the current working directory to the cloned repository.
func CloneRepository(t *testing.T, remoteDir string) {
	CloneRepositoryBranch(t, remoteDir, config.BranchMain)
}

// CloneRepositoryBranch clones the repository at the specified branch into
// a temporary directory and changes the current working directory to the cloned
// repository.
func CloneRepositoryBranch(t *testing.T, remoteDir, branch string) {
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	RunGit(t, "clone", "--branch", branch, remoteDir, ".")
	RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	configNewGitRepository(t)
}

// RunGit runs git with the specified arguments, aborting the test on any error.
func RunGit(t *testing.T, args ...string) {
	if err := command.Run(t.Context(), "git", args...); err != nil {
		t.Fatal(err)
	}
}
