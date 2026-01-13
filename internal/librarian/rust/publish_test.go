// Copyright 2026 Google LLC
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
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublishCratesSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	cfg := &config.Release{
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
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesWithNewCrate(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	cfg := &config.Release{
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
	_ = testhelper.SetupRepoWithChange(t, "release-with-new-crate")
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	files := []string{
		path.Join("src", "pubsub", "Cargo.toml"),
		path.Join("src", "pubsub", "src", "lib.rs"),
	}
	lastTag := "release-with-new-crate"
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesWithRootsPem(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	tmpDir := t.TempDir()
	rootsPem := path.Join(tmpDir, "roots.pem")
	if err := os.WriteFile(rootsPem, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Release{
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
	_ = testhelper.SetupRepoWithChange(t, "release-with-roots-pem")
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-with-roots-pem"
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesWithBadManifest(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	cfg := &config.Release{
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
	_ = testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelper.NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	name = path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("bad-toml = {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err == nil {
		t.Errorf("expected an error with a bad manifest file")
	}
}

func TestPublishCratesGetPlanError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "git",
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan generation")
	}
}

func TestPublishCratesPlanMismatchError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "echo")
	cfg := &config.Release{
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
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan comparison")
	}
}

func TestPublishCratesSkipSemverChecks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}

	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
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

	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": cargoScript,
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	// This should fail because semver-checks fails.
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err == nil {
		t.Fatal("expected an error from semver-checks")
	}
	// Skipping the checks should succeed.
	if err := publishCrates(t.Context(), cfg, true, false, true, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	cfg := &config.Release{
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
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)

	if err := Publish(t.Context(), cfg, true, false, false); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithLocalChangesError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
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
	remoteDir := testhelper.SetupRepoWithChange(t, "release-with-local-changes-error")
	testhelper.CloneRepository(t, remoteDir)
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	if err := command.Run(t.Context(), "git", "add", path.Join("src", "pubsub")); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: created pubsub", "."); err != nil {
		t.Fatal(err)
	}
	if err := Publish(t.Context(), config, true, false, false); err == nil {
		t.Errorf("expected an error publishing with unpushed local commits")
	}
}

func TestPublishPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := Publish(t.Context(), config, true, false, false); err == nil {
		t.Errorf("expected a preflight error with a bad git command")
	}
}

func TestPublishLastTagError(t *testing.T) {
	const echo = "/bin/echo"
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, echo)
	config := config.Release{
		Remote: "origin",
		Branch: "invalid-branch",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	if err := Publish(t.Context(), &config, true, false, false); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}

func TestPublishCratesDryRunKeepGoing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}

	testhelper.RequireCommand(t, "git")
	tmpDir := t.TempDir()
	// Create a fake cargo that captures its arguments.
	cargoScript := path.Join(tmpDir, "cargo")
	script := `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
elif [ "$1" == "workspaces" ] && [ "$2" == "publish" ]; then
	echo $@ >> "` + path.Join(tmpDir, "cargo_args.txt") + `"
else
	/bin/echo $@
fi
`
	if err := os.WriteFile(cargoScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": cargoScript,
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	if err := publishCrates(t.Context(), cfg, true, true, false, lastTag, files); err != nil {
		t.Fatal(err)
	}

	// Verify that arguments were passed to cargo workspaces publish.
	output, err := os.ReadFile(path.Join(tmpDir, "cargo_args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "--keep-going") {
		t.Errorf("expected cargo command to contain '--keep-going', got: %s", string(output))
	}
	if count := strings.Count(string(output), "--dry-run"); count != 1 {
		t.Errorf("expected cargo command to contain '--dry-run' once, but found %d times: %s", count, string(output))
	}
}

func TestPublishCratesSemverChecksKeepGoing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}

	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
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

	cfg := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": cargoScript,
		},
	}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	// This should fail because semver-checks fails.
	if err := publishCrates(t.Context(), cfg, true, false, false, lastTag, files); err == nil {
		t.Fatal("expected an error from semver-checks")
	}
	// With --keep-going, this should succeed.
	if err := publishCrates(t.Context(), cfg, true, true, false, lastTag, files); err != nil {
		t.Fatal(err)
	}
}
