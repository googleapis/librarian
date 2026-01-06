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
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestCargoPreFlightSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	tools := []config.Tool{
		{Name: "cargo-semver-checks"},
	}
	if err := cargoPreFlight(context.Background(), "cargo", tools); err != nil {
		t.Fatal(err)
	}
}

func TestCargoPreFlightBadCargo(t *testing.T) {
	tools := []config.Tool{
		{Name: "cargo-semver-checks"},
	}
	if err := cargoPreFlight(context.Background(), "not-a-valid-cargo", tools); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestCargoPreFlightBadTool(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	tools := []config.Tool{
		{Name: "not-a-valid-tool", Version: "0.0.1"},
	}
	if err := cargoPreFlight(context.Background(), "cargo", tools); err == nil {
		t.Error("expected an error, got none")
	}
}

func TestPreflightMissingGit(t *testing.T) {
	if err := PreFlight(t.Context(), map[string]string{"git": "git-is-not-installed"}, "", nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreflightMissingCargo(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	if err := PreFlight(t.Context(), map[string]string{"cargo": "cargo-is-not-installed"}, "", nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreflightMissingUpstream(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	preinstalled := map[string]string{
		"cargo": "/bin/echo",
	}
	testhelper.ContinueInNewGitRepository(t, t.TempDir())
	if err := PreFlight(t.Context(), preinstalled, "origin", nil); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestPreflightWithTools(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "/bin/echo")
	preinstalled := map[string]string{
		"cargo": "/bin/echo",
	}
	tools := []config.Tool{
		{
			Name:    "cargo-semver-checks",
			Version: "0.42.0",
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := PreFlight(t.Context(), preinstalled, "origin", tools); err != nil {
		t.Errorf("expected a successful run, got=%v", err)
	}
}

func TestPreflightToolFailure(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	preinstalled := map[string]string{
		// Using `git install blah blah` will fail.
		"cargo": "git",
	}
	tools := []config.Tool{
		{
			Name:    "invalid-tool-name---",
			Version: "a.b.c",
		},
	}
	testhelper.SetupForVersionBump(t, "test-preflight-with-tools")
	if err := PreFlight(t.Context(), preinstalled, "origin", tools); err == nil {
		t.Errorf("expected an error installing cargo-semver-checks")
	}
}
