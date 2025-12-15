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

package librarian

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
)

func TestPreflightMissingGit(t *testing.T) {
	release := &config.Release{
		Preinstalled: map[string]string{
			"git": "git-is-not-installed",
		},
	}
	if err := preflight(t.Context(), "rust", release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingCargo(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	release := &config.Release{
		Preinstalled: map[string]string{
			"cargo": "cargo-is-not-installed",
		},
	}
	if err := preflight(t.Context(), "rust", release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightMissingUpstream(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	release := &config.Release{
		Preinstalled: map[string]string{
			"cargo": "/bin/echo",
		},
		Remote: "upstream",
	}
	testhelpers.ContinueInNewGitRepository(t, t.TempDir())
	if err := preflight(t.Context(), "rust", release); err == nil {
		t.Fatal(err)
	}
}

func TestPreflightWithTools(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	testhelpers.RequireCommand(t, "/bin/echo")
	release := &config.Release{
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
	testhelpers.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := preflight(t.Context(), "rust", release); err != nil {
		t.Errorf("expected a successful run, got=%v", err)
	}
}

func TestPreflightToolFailure(t *testing.T) {
	testhelpers.RequireCommand(t, "git")
	release := &config.Release{
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
	testhelpers.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := preflight(t.Context(), "rust", release); err == nil {
		t.Errorf("expected an error installing cargo-semver-checks")
	}
}
