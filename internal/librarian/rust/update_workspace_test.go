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
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/testhelper"
)

func TestUpdateWorkspace(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")

	workspaceDir := t.TempDir()
	workspaceCargo := `[workspace]
members = ["test-lib"]
resolver = "2"

[workspace.package]
edition = "2024"
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), []byte(workspaceCargo), 0644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(workspaceDir, "test-lib")
	srcDir := filepath.Join(libDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	libCargo := `[package]
name = "test-lib"
version = "0.1.0"
edition.workspace = true
`
	if err := os.WriteFile(filepath.Join(libDir, "Cargo.toml"), []byte(libCargo), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "lib.rs"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workspaceDir)
	if err := UpdateWorkspace(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "Cargo.lock")); err != nil {
		t.Errorf("Cargo.lock not created: %v", err)
	}
}
