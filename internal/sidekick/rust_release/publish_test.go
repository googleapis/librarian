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
	"runtime"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/config"
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
	if err := Publish(config, true, false); err != nil {
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
	if err := command.Run("git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	cloneRepository(t, remoteDir)
	if err := Publish(config, true, false); err != nil {
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
	if err := Publish(config, true, false); err != nil {
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
	if err := command.Run("git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run("git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	if err := Publish(config, true, false); err == nil {
		t.Errorf("expected an error publishing a dirty local repository")
	}
}

func TestPublishPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := Publish(config, true, false); err == nil {
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
	if err := Publish(&config, true, false); err == nil {
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
	if err := command.Run("git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	cloneRepository(t, remoteDir)
	if err := Publish(config, true, false); err == nil {
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
	if err := Publish(config, true, false); err == nil {
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
	if err := Publish(config, true, false); err == nil {
		t.Fatalf("expected an error during plan comparison")
	}
}

func TestPublishSkipSemverChecks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}

	requireCommand(t, "git")
	requireCommand(t, "/bin/echo")
	tmpDir := t.TempDir()
	// Create a fake cargo that fails on `semver-checks`
	cargoScript := path.Join(tmpDir, "cargo")
	script := `#!/bin/bash
if [ "$1" == "semver-checks" ]; then
	exit 1
elif [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	/bin/echo $@
fi
`
	if err := os.WriteFile(cargoScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": cargoScript,
		},
	}
	remoteDir := setupForPublish(t, "release-2001-02-03")
	cloneRepository(t, remoteDir)

	// This should fail because semver-checks fails.
	if err := Publish(config, true, false); err == nil {
		t.Fatal("expected an error from semver-checks")
	}

	// Skipping the checks should succeed.
	if err := Publish(config, true, true); err != nil {
		t.Fatal(err)
	}
}
