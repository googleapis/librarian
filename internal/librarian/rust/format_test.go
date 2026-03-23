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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "rustfmt")

	workspaceDir := t.TempDir()
	libName := "format-test-lib"
	libDir := filepath.Join(workspaceDir, libName)
	srcDir := filepath.Join(libDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write workspace Cargo.toml.
	workspaceCargo := `[workspace]
members = ["` + libName + `"]
resolver = "2"

[workspace.package]
edition = "2024"
`
	if err := os.WriteFile(filepath.Join(workspaceDir, "Cargo.toml"), []byte(workspaceCargo), 0644); err != nil {
		t.Fatal(err)
	}

	// Write library Cargo.toml with inconsistent formatting.
	libCargo := `[package]
name    =   "` + libName + `"
version =    "0.1.0"
edition.workspace  =    true
`
	if err := os.WriteFile(filepath.Join(libDir, "Cargo.toml"), []byte(libCargo), 0644); err != nil {
		t.Fatal(err)
	}

	// Write unformatted Rust source.
	unformatted := `fn   main(  )   {   println!(  "hello"  )  ;   }
`
	if err := os.WriteFile(filepath.Join(srcDir, "lib.rs"), []byte(unformatted), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(workspaceDir)

	library := &config.Library{
		Name:   libName,
		Output: libDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	// Verify Cargo.toml was formatted by taplo (extra spaces removed).
	got, err := os.ReadFile(filepath.Join(libDir, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[package]
name = "` + libName + `"
version = "0.1.0"
edition.workspace = true
`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify Rust source was formatted by cargo fmt.
	gotRs, err := os.ReadFile(filepath.Join(srcDir, "lib.rs"))
	if err != nil {
		t.Fatal(err)
	}
	wantRs := `fn main() {
    println!("hello");
}
`
	if diff := cmp.Diff(wantRs, string(gotRs)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
