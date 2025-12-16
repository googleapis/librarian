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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

func TestBumpVersionsSuccess(t *testing.T) {
	testhelpers.RequireCommand(t, "/bin/echo")
	testhelpers.RequireCommand(t, "git")
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
	testhelpers.SetupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelpers.NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(t.Context(), config); err != nil {
		t.Fatal(err)
	}
}
func TestBumpVersionsNoCargoTools(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
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
	testhelpers.SetupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelpers.NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(t.Context(), config); err != nil {
		t.Fatal(err)
	}
}
func TestBumpVersionsNoSemverChecks(t *testing.T) {
	testhelpers.RequireCommand(t, "/bin/echo")
	testhelpers.RequireCommand(t, "git")
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
	testhelpers.SetupForVersionBump(t, "release-2001-02-03")
	name := path.Join("src", "storage", "src", "lib.rs")
	if err := os.WriteFile(name, []byte(testhelpers.NewLibRsContents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: changed storage", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(t.Context(), config); err != nil {
		t.Fatal(err)
	}
}

func TestBumpVersionsPreflightError(t *testing.T) {
	config := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-not-found",
		},
	}
	if err := BumpVersions(t.Context(), config); err == nil {
		t.Errorf("expected an error in BumpVersions() with a bad git command")
	}
}

func TestBumpVersionsLastTagError(t *testing.T) {
	const echo = "/bin/echo"
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, echo)
	config := config.Release{
		Remote: "origin",
		Branch: "invalid-branch",
		Preinstalled: map[string]string{
			"cargo": echo,
		},
	}
	testhelpers.SetupForVersionBump(t, "last-tag-error")
	if err := BumpVersions(t.Context(), &config); err == nil {
		t.Fatalf("expected an error during GetLastTag")
	}
}

func TestBumpVersionsManifestError(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	config := &config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"git":   "git",
			"cargo": "git",
		},
	}
	testhelpers.SetupForVersionBump(t, "release-bad-manifest")
	name := path.Join("src", "storage", "Cargo.toml")
	if err := os.WriteFile(name, []byte("invalid-toml-file = {"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", "feat: broke storage manifest file", "."); err != nil {
		t.Fatal(err)
	}
	if err := BumpVersions(t.Context(), config); err == nil {
		t.Errorf("expected error while processing invalid manifest file")
	}
}
