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
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPreflightMissingGit(t *testing.T) {
	release := config.Release{
		Preinstalled: map[string]string{
			"git": "git-is-not-installed",
		},
	}
	if err := PreFlight(t.Context(), &release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingCargo(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	release := config.Release{
		Preinstalled: map[string]string{
			"cargo": "cargo-is-not-installed",
		},
	}
	if err := PreFlight(t.Context(), &release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingUpstream(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	release := config.Release{
		Preinstalled: map[string]string{
			"cargo": "/bin/echo",
		},
		Remote: "origin",
	}
	testhelper.ContinueInNewGitRepository(t, t.TempDir())
	if err := PreFlight(t.Context(), &release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightWithTools(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			"cargo": "/bin/echo",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{
					Name:    "cargo-semver-checks",
					Version: "0.42.0",
				},
			},
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := PreFlight(t.Context(), &release); err != nil {
		t.Errorf("expected a successful run, got=%v", err)
	}
}

func TestPreflightToolFailure(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	release := config.Release{
		Remote: "origin",
		Branch: "main",
		Preinstalled: map[string]string{
			// Using `git install blah blah` will fail.
			"cargo": "git",
		},
		Tools: map[string][]config.Tool{
			"cargo": {
				{
					Name:    "invalid-tool-name---",
					Version: "a.b.c",
				},
			},
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := PreFlight(t.Context(), &release); err == nil {
		t.Errorf("expected an error installing cargo-semver-checks")
	}
}

func TestGitExe(t *testing.T) {
	release := config.Release{}
	if got := gitExe(&release); got != "git" {
		t.Errorf("mismatch in gitExe(), want=git, got=%s", got)
	}
	release = config.Release{
		Preinstalled: map[string]string{
			"git": "alternative",
		},
	}
	if got := gitExe(&release); got != "alternative" {
		t.Errorf("mismatch in gitExe(), want=alternative, got=%s", got)
	}
}

func TestCargoExe(t *testing.T) {
	release := config.Release{}
	if got := cargoExe(&release); got != "cargo" {
		t.Errorf("mismatch in cargoExe(), want=cargo, got=%s", got)
	}
	release = config.Release{
		Preinstalled: map[string]string{
			"cargo": "alternative",
		},
	}
	if got := cargoExe(&release); got != "alternative" {
		t.Errorf("mismatch in cargoExe(), want=alternative, got=%s", got)
	}
}
