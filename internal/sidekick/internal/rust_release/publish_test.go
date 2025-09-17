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

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/external"
)

func TestPublishSuccess(t *testing.T) {
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
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := setupForPublish(t, "release-2001-02-03")
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithNewCrate(t *testing.T) {
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
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := setupForPublish(t, "release-with-new-crate")
	addCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := external.Run("git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithRootsPem(t *testing.T) {
	requireCommand(t, "git")
	requireCommand(t, "/bin/echo")
	tmpDir := t.TempDir()
	rootsPem := path.Join(tmpDir, "roots.pem")
	if err := os.WriteFile(rootsPem, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
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
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
		RootsPem: rootsPem,
	}
	remoteDir := setupForPublish(t, "release-with-roots-pem")
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithLocalChangesError(t *testing.T) {
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
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := setupForPublish(t, "release-with-local-changes-error")
	cloneRepository(t, remoteDir)
	addCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := external.Run("git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	if err := Publish(config, true); err == nil {
		t.Errorf("expected an error publishing a dirty local repository")
	}
}

func TestPublishPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := Publish(config, true); err == nil {
		t.Errorf("expected an error in BumpVersions() with a bad git command")
	}
}

func TestPublishLastTagError(t *testing.T) {
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
	remoteDir := setupForPublish(t, "release-2001-02-03")
	cloneRepository(t, remoteDir)
	if err := Publish(&config, true); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}

func TestPublishBadManifest(t *testing.T) {
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
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := setupForPublish(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(newLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	name = path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("bad-toml = {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := external.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err == nil {
		t.Errorf("expected an error with a bad manifest file")
	}
}

func TestPublishGetPlanError(t *testing.T) {
	requireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "git",
		},
	}
	remoteDir := setupForPublish(t, "release-2001-02-03")
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err == nil {
		t.Fatalf("expected an error during plan generation")
	}
}

func TestPublishPlanMismatchError(t *testing.T) {
	requireCommand(t, "git")
	requireCommand(t, "echo")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	remoteDir := setupForPublish(t, "release-2001-02-03")
	cloneRepository(t, remoteDir)
	if err := Publish(config, true); err == nil {
		t.Fatalf("expected an error during plan comparison")
	}
}
