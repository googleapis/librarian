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
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublishCratesSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	exit 0
fi
`)
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesWithNewCrate(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-pubsub"
else
	exit 0
fi
`)
	_ = testhelper.SetupRepoWithChange(t, "release-with-new-crate")
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	testhelper.RunGit(t, "add", path.Join("src", "pubsub"))
	testhelper.RunGit(t, "commit", "-m", "feat: created pubsub", ".")
	files := []string{
		path.Join("src", "pubsub", "Cargo.toml"),
		path.Join("src", "pubsub", "src", "lib.rs"),
	}
	lastTag := "release-with-new-crate"
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesWithBadManifest(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
exit 0
`)
	_ = testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelper.NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	name = path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("bad-toml = {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "commit", "-m", "feat: changed storage", ".")
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err == nil {
		t.Errorf("expected an error with a bad manifest file")
	}
}

func TestPublishCratesGetPlanError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{}
	setupFakeCargoScript(t, `#!/bin/bash
exit 1
`)
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan generation")
	}
}

func TestPublishCratesPlanMismatchError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "unplanned-crate"
else
	exit 0
fi
`)
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err == nil {
		t.Fatalf("expected an error during plan comparison")
	}
}

func TestPublishCratesSkipSemverChecks(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	script := `#!/bin/bash
if [ "$1" == "semver-checks" ]; then
	exit 1
elif [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	exit 0
fi
`
	setupFakeCargoScript(t, script)

	cfg := &config.Config{}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	// This should fail because semver-checks fails.
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err == nil {
		t.Fatal("expected an error from semver-checks")
	}
	// Skipping the checks should succeed.
	if err := publishCrates(t.Context(), PublishParams{
		Config:           cfg,
		DryRun:           true,
		SkipSemverChecks: true,
	}, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	exit 0
fi
`)
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)

	if err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPublishWithLocalChangesError(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := &config.Config{
		Tools: &config.Tools{
			Cargo: []*config.CargoTool{
				{Name: "cargo-semver-checks", Version: "1.2.3"},
				{Name: "cargo-workspaces", Version: "3.4.5"},
			},
		},
	}
	setupFakeCargoScript(t, `#!/bin/bash
exit 0
`)
	remoteDir := testhelper.SetupRepoWithChange(t, "release-with-local-changes-error")
	testhelper.CloneRepository(t, remoteDir)
	testhelper.AddCrate(t, path.Join("src", "pubsub"), "google-cloud-pubsub")
	testhelper.RunGit(t, "add", path.Join("src", "pubsub"))
	testhelper.RunGit(t, "commit", "-m", "feat: created pubsub", ".")
	if err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}); err == nil {
		t.Errorf("expected an error publishing with unpushed local commits")
	}
}

func TestPublishPreflightError(t *testing.T) {
	cfg := &config.Config{}
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir) // empty path so git is not found
	if err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}); err == nil {
		t.Errorf("expected a preflight error with a bad git command")
	}
}

func TestPublishCratesDryRunKeepGoing(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	tmpDir := t.TempDir()
	// Create a fake cargo that captures its arguments.
	script := `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
elif [ "$1" == "workspaces" ] && [ "$2" == "publish" ]; then
	echo $@ >> "` + filepath.Join(tmpDir, "cargo_args.txt") + `"
else
	exit 0
fi
`
	setupFakeCargoScript(t, script)

	cfg := &config.Config{}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	if err := publishCrates(t.Context(), PublishParams{
		Config:          cfg,
		DryRun:          true,
		DryRunKeepGoing: true,
	}, lastTag, files); err != nil {
		t.Fatal(err)
	}

	// Verify that arguments were passed to cargo workspaces publish.
	output, err := os.ReadFile(filepath.Join(tmpDir, "cargo_args.txt"))
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
	testhelper.RequireCommand(t, "git")
	script := `#!/bin/bash
if [ "$1" == "semver-checks" ]; then
	exit 1
elif [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	exit 0
fi
`
	setupFakeCargoScript(t, script)

	cfg := &config.Config{}
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	testhelper.CloneRepository(t, remoteDir)
	files := []string{
		path.Join("src", "storage", "Cargo.toml"),
		path.Join("src", "storage", "src", "lib.rs"),
	}
	lastTag := "release-2001-02-03"

	// This should fail because semver-checks fails.
	if err := publishCrates(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	}, lastTag, files); err == nil {
		t.Fatal("expected an error from semver-checks")
	}
	// With --keep-going, this should succeed.
	if err := publishCrates(t.Context(), PublishParams{
		Config:          cfg,
		DryRun:          true,
		DryRunKeepGoing: true,
	}, lastTag, files); err != nil {
		t.Fatal(err)
	}
}

func TestPublishCratesValidation(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	// Create a fake cargo that ALWAYS plans "google-cloud-storage"
	script := `#!/bin/bash
if [ "$1" == "workspaces" ] && [ "$2" == "plan" ]; then
	echo "google-cloud-storage"
else
	exit 0
fi
`
	setupFakeCargoScript(t, script)

	cfg := &config.Config{}
	// Setup a dummy repo
	remoteDir := testhelper.SetupRepoWithChange(t, "test-validation")
	testhelper.CloneRepository(t, remoteDir)
	lastTag := "test-validation"

	for _, test := range []struct {
		name    string
		files   []string
		wantErr string
	}{
		{
			name: "exact match on storage",
			files: []string{
				path.Join("src", "storage", "Cargo.toml"),
				path.Join("src", "storage", "src", "lib.rs"),
			},
			wantErr: "",
		},
		{
			name: "subset with pubsub and storage",
			files: []string{
				path.Join("src", "storage", "Cargo.toml"),
				path.Join("src", "pubsub", "Cargo.toml"),
			},
			wantErr: "",
		},
		{
			name:    "superset missing storage change",
			files:   []string{},
			wantErr: "unplanned crate \"google-cloud-storage\" found in workspace plan",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := publishCrates(t.Context(), PublishParams{
				Config:           cfg,
				DryRun:           true,
				SkipSemverChecks: true,
			}, lastTag, test.files)
			var got string
			if err != nil {
				got = err.Error()
			}
			if diff := cmp.Diff(test.wantErr, got); diff != "" {
				t.Errorf("publishCrates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunSemverChecks(t *testing.T) {
	for _, test := range []struct {
		name            string
		manifests       map[string]string
		dryRunKeepGoing bool
	}{
		{
			name: "all crates pass",
			manifests: map[string]string{
				"crate-a": "a/Cargo.toml",
				"crate-b": "b/Cargo.toml",
			},
		},
		{
			name: "dry-run-keep-going ignores failures",
			manifests: map[string]string{
				"fail-me": "fail/Cargo.toml",
			},
			dryRunKeepGoing: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			script := `#!/bin/bash
if [[ "$*" == *"fail-me"* ]]; then
	exit 1
fi
exit 0
`
			setupFakeCargoScript(t, script)
			sData := semverData{
				manifests:       test.manifests,
				dryRunKeepGoing: test.dryRunKeepGoing,
			}

			if err := runSemverChecks(t.Context(), sData); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRunSemverChecks_Errors(t *testing.T) {
	manifests := map[string]string{
		"fail-me": "fail/Cargo.toml",
	}
	wantErr := errSemverCheck

	script := `#!/bin/bash
exit 1
`
	setupFakeCargoScript(t, script)
	sData := semverData{
		manifests: manifests,
	}
	err := runSemverChecks(t.Context(), sData)
	if err == nil {
		t.Error("runSemverChecks() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("runSemverChecks() error = %v, want to contain %v", err, wantErr)
	}
}

// setupFakeCargoScript writes a shell script named 'cargo' to a temporary
// directory and prepends it to PATH.
func setupFakeCargoScript(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	tmpDir := t.TempDir()
	cargoScript := filepath.Join(tmpDir, "cargo")
	if err := os.WriteFile(cargoScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
