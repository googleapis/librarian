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
	"path/filepath"
	"strings"
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestCreate(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "taplo")
	testhelper.RequireCommand(t, "git")

	t.Chdir(t.TempDir())
	if err := cmdtest.Run(t.Context(), "git", "init"); err != nil {
		t.Fatal(err)
	}

	workspaceCargo := `
[workspace]
members = []
`
	if err := os.WriteFile("Cargo.toml", []byte(workspaceCargo), 0644); err != nil {
		t.Fatal(err)
	}

	const libName = "secretmanager"
	if err := Create(t.Context(), libName); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		// `cargo new --vcs none --lib` creates a Cargo.toml with the library
		// name and a default src/lib.rs file.
		{filepath.Join(libName, "Cargo.toml"), `name = "secretmanager"`},
		{filepath.Join(libName, "src", "lib.rs"), "pub fn add(left: u64, right: u64) -> u64 {"},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), test.want) {
				t.Errorf("%q missing expected string: %q\ngot:\n%s", test.path, test.want, string(got))
			}
		})
	}
}
