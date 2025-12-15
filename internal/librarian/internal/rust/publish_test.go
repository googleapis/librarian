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

package rust

import (
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

func newCargoMock(t *testing.T, plannedCrates []string) string {
	t.Helper()
	tmpDir := t.TempDir()
	cargoScript := path.Join(tmpDir, "cargo")
	script := `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "` + strings.Join(plannedCrates, "\n") + `"
elif [ "$1" == "semver-checks" ]; then
    exit 0 # success
else
	/bin/echo $@
fi
`
	if err := os.WriteFile(cargoScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return cargoScript
}

func TestPublishSuccess(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	cargoMock := newCargoMock(t, []string{"google-cloud-storage"})
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoMock,
			},
			Tools: map[string][]config.Tool{
				"cargo": {
					{Name: "cargo-semver-checks", Version: "1.2.3"},
					{Name: "cargo-workspaces", Version: "3.4.5"},
				},
			},
		},
	}
	lastTag := "release-2001-02-03"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithNewCrate(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	cargoMock := newCargoMock(t, []string{"google-cloud-storage", "google-cloud-pubsub"})
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoMock,
			},
			Tools: map[string][]config.Tool{
				"cargo": {
					{Name: "cargo-semver-checks", Version: "1.2.3"},
					{Name: "cargo-workspaces", Version: "3.4.5"},
				},
			},
		},
	}
	lastTag := "release-with-new-crate"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "pubsub", "Cargo.toml"),
	}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithRootsPem(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	tmpDir := t.TempDir()
	rootsPem := path.Join(tmpDir, "roots.pem")
	if err := os.WriteFile(rootsPem, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	cargoMock := newCargoMock(t, []string{"google-cloud-storage"})
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoMock,
			},
			Tools: map[string][]config.Tool{
				"cargo": {
					{Name: "cargo-semver-checks", Version: "1.2.3"},
					{Name: "cargo-workspaces", Version: "3.4.5"},
				},
			},
			RootsPem: rootsPem,
		},
	}
	lastTag := "release-with-roots-pem"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishPreflightError(t *testing.T) {
	cfg := &config.Config{
		Release: &config.Release{
			Preinstalled: map[string]string{
				"cargo": "cargo-not-found",
			},
		},
	}
	if err := PreFlight(t.Context(), cfg.Release); err == nil {
		t.Errorf("expected an error in PreFlight() with a bad cargo command")
	}
}

func TestPublishBadManifest(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	cargoMock := newCargoMock(t, []string{"google-cloud-storage"})
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoMock,
			},
			Tools: map[string][]config.Tool{
				"cargo": {
					{Name: "cargo-semver-checks", Version: "1.2.3"},
					{Name: "cargo-workspaces", Version: "3.4.5"},
				},
			},
		},
	}
	lastTag := "release-2001-02-03"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelpers.InitialCargoContents), 0644); err != nil {
		t.Fatal(err)
	}
	name = path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("bad-toml = {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err == nil {
		t.Errorf("expected an error with a bad manifest file")
	}
}

func TestPublishGetPlanError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": "git", // Using git to cause an error
			},
		},
	}
	lastTag := "release-2001-02-03"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan generation")
	}
}

func TestPublishPlanMismatchError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "echo")
	cargoMock := newCargoMock(t, []string{"other-crate"})
	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoMock,
			},
			Tools: map[string][]config.Tool{
				"cargo": {
					{Name: "cargo-semver-checks", Version: "1.2.3"},
					{Name: "cargo-workspaces", Version: "3.4.5"},
				},
			},
		},
	}
	lastTag := "release-2001-02-03"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan comparison")
	}
}

func TestPublishSkipSemverChecks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}

	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
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

	cfg := &config.Config{
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
			Preinstalled: map[string]string{
				"git":   "git",
				"cargo": cargoScript,
			},
		},
	}
	lastTag := "release-2001-02-03"
	remoteDir := testhelpers.SetupForPublish(t, lastTag)
	testhelpers.CloneRepository(t, remoteDir)
	files := []string{path.Join("src", "storage", "Cargo.toml")}

	// This should fail because semver-checks fails.
	if err := Publish(t.Context(), cfg, true, false, lastTag, files); err == nil {
		t.Fatal("expected an error from semver-checks")
	}

	// Skipping the checks should succeed.
	if err := Publish(t.Context(), cfg, true, true, lastTag, files); err != nil {
		t.Fatal(err)
	}
}
