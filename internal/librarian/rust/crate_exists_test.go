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
	"fmt"
	"os"
	"testing"
)

const formatTestCargoToml = `
[workspace]
resolver = "3"

members = [
 "src/auth",
%s
]
`

func TestCrateExists(t *testing.T) {
	const testDir = "src/generated/test/service/v1"
	for _, test := range []struct {
		name       string
		cargoText  string
		makeDir    string
		wantResult bool
	}{
		{
			name:       "exists",
			cargoText:  fmt.Sprintf(`  "%s",`, testDir),
			makeDir:    testDir,
			wantResult: true,
		},
		{
			name:       "not exists",
			cargoText:  `  "src/generated/test/service/v2",`,
			makeDir:    "src/generated/test/service/v2",
			wantResult: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := t.TempDir()
			t.Chdir(d)
			if err := os.MkdirAll(test.makeDir, 0775); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("Cargo.toml", fmt.Appendf(nil, formatTestCargoToml, test.cargoText), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := crateExists(testDir)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantResult != got {
				t.Errorf("crateExists() mismatch, want=%v, got=%v", test.wantResult, got)
			}
		})
	}
}

func TestCrateExists_Errors(t *testing.T) {
	const testDir = "src/generated/test/service/v1"
	for _, test := range []struct {
		name      string
		cargoText string
		makeDir   string
	}{
		{
			name:      "only directory",
			cargoText: `  "bad-bad-bad",`,
			makeDir:   testDir,
		},
		{
			name:      "only entry",
			cargoText: fmt.Sprintf(`  "%s",`, testDir),
			makeDir:   "src/generated",
		},
		{
			name:      "entry is substring",
			cargoText: `  "src/generated/test/service/v1/weird/but/should/test",`,
			makeDir:   testDir,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := t.TempDir()
			t.Chdir(d)
			if err := os.MkdirAll(test.makeDir, 0775); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("Cargo.toml", fmt.Appendf(nil, formatTestCargoToml, test.cargoText), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := crateExists(testDir)
			if err == nil {
				t.Fatalf("crateExists() succeed when an error was expected: %v", got)
			}
		})
	}
}

func TestCrateExistsDirectoryError(t *testing.T) {
	d := t.TempDir()
	t.Chdir(d)
	// Create part of the path, but make it unreadable.
	if err := os.MkdirAll("src", 0000); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("Cargo.toml", []byte("# test only"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := crateExists("src/generated/unreachable"); err == nil {
		t.Errorf("expected an error from crateExists(), got=%v", got)
	}
}

func TestCrateExistsCargoTomlError(t *testing.T) {
	d := t.TempDir()
	t.Chdir(d)
	// Create part of the path, but make it unreadable.
	if err := os.MkdirAll("src/generated/test", 0755); err != nil {
		t.Fatal(err)
	}
	if got, err := crateExists("src/generated/test"); err == nil {
		t.Errorf("expected an error from crateExists(), got=%v", got)
	}
}
