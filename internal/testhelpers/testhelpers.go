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

// Package testhelpers provides helper functions for tests.
package testhelpers

import (
	"context"
	"os/exec"
	"testing"

	"github.com/googleapis/librarian/internal/command"
)

// RequireCommand skips the test if the specified command is not found in PATH.
// Use this to skip tests that depend on external tools like protoc, cargo, or
// taplo, so that `go test ./...` will always pass on a fresh clone of the
// repo.
func RequireCommand(t *testing.T, cmd string) {
	t.Helper()
	if _, err := exec.LookPath(cmd); err != nil {
		t.Skipf("skipping test because %s is not installed", cmd)
	}
}

// SetupRepo creates a temporary git repository for testing.
// It initializes a git repository, sets up a remote, and creates an initial commit with a tag.
func SetupRepo(t *testing.T, tag string) {
	t.Helper()
	ctx := context.Background()
	remoteDir := t.TempDir()
	if err := command.Run(ctx, "git", "init", "--bare", remoteDir); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "clone", remoteDir, "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "commit", "--allow-empty", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "tag", tag); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "push", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(ctx, "git", "push", "origin", tag); err != nil {
		t.Fatal(err)
	}
}
