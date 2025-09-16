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
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

const (
	initialCargoContents = `# Example Cargo file
[package]
name    = "%s"
version = "1.0.0"
`

	initialLibRsContents = `pub fn test() -> &'static str { "Hello World" }`
)

func TestBumpVersionsSuccess(t *testing.T) {
	requireCommand(t, "git")
	requireCommand(t, "/bin/echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{Name: "cargo-semver-checks", Version: "1.2.3"},
			},
		},
	}
	setupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(config); err != nil {
		t.Fatal(err)
	}
}

func TestBumpVersionsNoCargoTools(t *testing.T) {
	requireCommand(t, "git")
	requireCommand(t, "/bin/echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"not-cargo": {
				{Name: "semver-checks", Version: "1.2.3"},
			},
		},
	}
	setupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(config); err != nil {
		t.Fatal(err)
	}
}

func TestBumpVersionsNoSemverChecks(t *testing.T) {
	requireCommand(t, "git")
	requireCommand(t, "/bin/echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{Name: "some-other-tool", Version: "1.2.3"},
			},
		},
	}
	setupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(config); err != nil {
		t.Fatal(err)
	}
}

func TestBumpVersionsPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := BumpVersions(config); err == nil {
		t.Errorf("expected an error in BumpVersions() with a bad git command")
	}
}

func TestBumpVersionsLastTagError(t *testing.T) {
	const echo = "/bin/echo"
	requireCommand(t, "git")
	requireCommand(t, echo)
	config := config.Release{
		Remote: "origin",
		Branch: "invalid-branch",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	setupForVersionBump(t, "last-tag-error")
	if err := BumpVersions(&config); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}

func TestBumpVersionsManifestError(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "git",
		},
	}
	setupForVersionBump(t, "release-bad-manifest")
	name := path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("invalid-toml-file = {"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: broke storage manifest file", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(config); err == nil {
		t.Errorf("expected error while processing invalid manifest file")
	}
}

func setupForVersionBump(t *testing.T, wantTag string) {
	remoteDir := t.TempDir()
	continueInNewGitRepository(t, remoteDir)
	initRepositoryContents(t)
	if err := external.Run("git", "tag", wantTag); err != nil {
		t.Fatal(err)
	}
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	if err := external.Run("git", "clone", remoteDir, "."); err != nil {
		t.Fatal(err)
	}
	configNewGitRepository(t)
}

func continueInNewGitRepository(t *testing.T, tmpDir string) {
	t.Helper()
	requireCommand(t, "git")
	t.Chdir(tmpDir)
	if err := external.Run("git", "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	configNewGitRepository(t)
}

func requireCommand(t *testing.T, command string) {
	t.Helper()
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("skipping test because %s is not installed", command)
	}
}

func configNewGitRepository(t *testing.T) {
	if err := external.Run("git", "config", "user.email", "test@test-only.com"); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "config", "user.name", "Test Account"); err != nil {
		t.Fatal(err)
	}
}

func initRepositoryContents(t *testing.T) {
	t.Helper()
	requireCommand(t, "git")
	if err := os.WriteFile("README.md", []byte("# Empty Repo"), 0644); err != nil {
		t.Fatal(err)
	}
	addCrate(t, path.Join("src", "storage"), "google-cloud-storage")
	addCrate(t, path.Join("src", "gax-internal"), "google-cloud-gax-internal")
	addCrate(t, path.Join("src", "gax-internal", "echo-server"), "echo-server")
	addGeneratedCrate(t, path.Join("src", "generated", "cloud", "secretmanager", "v1"), "google-cloud-secretmanager-v1")
	if err := external.Run("git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "initial version"); err != nil {
		t.Fatal(err)
	}
}

func addGeneratedCrate(t *testing.T, location, name string) {
	t.Helper()
	addCrate(t, location, name)
	if err := os.WriteFile(path.Join(location, ".sidekick.toml"), []byte("# initial version"), 0644); err != nil {
		t.Fatal(err)
	}
}

func addCrate(t *testing.T, location, name string) {
	t.Helper()
	_ = os.MkdirAll(path.Join(location, "src"), 0755)
	contents := []byte(fmt.Sprintf(initialCargoContents, name))
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
