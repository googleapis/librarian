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

package dart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const initialPubspec = `name: %s
version: 0.4.0
dependencies:
  sdk: ^3.9.0
%s`

func TestBump(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	// Use relative paths so they are created within the git repository
	outputA := filepath.Join("packages", "lib_a")
	outputB := filepath.Join("packages", "lib_b")
	outputC := filepath.Join("packages", "lib_c")

	cfg := &config.Config{
		Language: "dart",
		Default: &config.Default{
			TagFormat: "{name}-v{version}",
			Output:    "packages",
		},
		Libraries: []*config.Library{
			{
				Name:    "lib_a",
				Version: "0.4.0",
				Output:  outputA,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:googleapis_auth": "^2.0.0",
					},
				},
			},
			{
				Name:    "lib_b",
				Version: "0.4.0",
				Output:  outputB,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:lib_a": "^0.4.0",
					},
				},
			},
			{
				Name:    "lib_c",
				Version: "0.4.0",
				Output:  outputC,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:lib_b": "^0.4.0",
					},
				},
			},
		},
	}

	// Set up git repo and simulate change in lib_a
	opts := testhelper.SetupOptions{
		Clone:  true,
		Config: cfg,
		Tags: []string{
			"lib_a-v0.4.0",
			"lib_b-v0.4.0",
			"lib_c-v0.4.0",
		},
		WithChanges: []string{
			filepath.Join(outputA, "lib", "a.dart"),
		},
	}

	// Write the pubspec.yaml files during test setup before tagging and commits
	testhelper.Setup(t, opts)

	writePubspec := func(path, name, depsSection string) {
		content := []byte(fmt.Sprintf(initialPubspec, name, depsSection))
		if err := os.WriteFile(filepath.Join(path, "pubspec.yaml"), content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(outputA, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outputB, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outputC, "lib"), 0755); err != nil {
		t.Fatal(err)
	}

	writePubspec(outputA, "lib_a", "")
	writePubspec(outputB, "lib_b", "  lib_a: ^0.4.0")
	writePubspec(outputC, "lib_c", "  lib_b: ^0.4.0")

	// Run Bump
	if err := Bump(t.Context(), cfg, true, "", "", "git"); err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	// Check in-memory updates
	wantVersions := map[string]string{
		"lib_a": "0.5.0",
		"lib_b": "0.5.0",
		"lib_c": "0.5.0",
	}
	for _, lib := range cfg.Libraries {
		if got := lib.Version; got != wantVersions[lib.Name] {
			t.Errorf("Library %s version: got %s, want %s", lib.Name, got, wantVersions[lib.Name])
		}
	}

	// Check pubspec.yaml outputs
	verifyPubspec := func(path, expectedVersion, expectedDeps string) {
		content, err := os.ReadFile(filepath.Join(path, "pubspec.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if !strings.Contains(text, "version: "+expectedVersion) {
			t.Errorf("Pubspec at %s missing version %s, content: %s", path, expectedVersion, text)
		}
		if expectedDeps != "" && !strings.Contains(text, expectedDeps) {
			t.Errorf("Pubspec at %s missing dependency section %q, content: %s", path, expectedDeps, text)
		}
	}

	verifyPubspec(outputA, "0.5.0", "")
	verifyPubspec(outputB, "0.5.0", "  lib_a: ^0.5.0")
	verifyPubspec(outputC, "0.5.0", "  lib_b: ^0.5.0")
}

func TestBump_ExplicitLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	outputA := filepath.Join("packages", "lib_a")
	outputB := filepath.Join("packages", "lib_b")

	cfg := &config.Config{
		Language: "dart",
		Default: &config.Default{
			TagFormat: "{name}-v{version}",
			Output:    "packages",
		},
		Libraries: []*config.Library{
			{
				Name:    "lib_a",
				Version: "0.4.0",
				Output:  outputA,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:googleapis_auth": "^2.0.0",
					},
				},
			},
			{
				Name:    "lib_b",
				Version: "0.4.0",
				Output:  outputB,
				Dart: &config.DartPackage{
					Packages: map[string]string{
						"package:lib_a": "^0.4.0",
					},
				},
			},
		},
	}

	opts := testhelper.SetupOptions{
		Clone:  true,
		Config: cfg,
		Tags: []string{
			"lib_a-v0.4.0",
			"lib_b-v0.4.0",
		},
	}
	testhelper.Setup(t, opts)

	writePubspec := func(path, name, depsSection string) {
		content := []byte(fmt.Sprintf(initialPubspec, name, depsSection))
		if err := os.WriteFile(filepath.Join(path, "pubspec.yaml"), content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(outputA, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outputB, "lib"), 0755); err != nil {
		t.Fatal(err)
	}

	writePubspec(outputA, "lib_a", "")
	writePubspec(outputB, "lib_b", "  lib_a: ^0.4.0")

	// Bump explicitly lib_a with a version override
	if err := Bump(t.Context(), cfg, false, "lib_a", "1.0.0", "git"); err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if diff := cmp.Diff("1.0.0", cfg.Libraries[0].Version); diff != "" {
		t.Errorf("lib_a version mismatch (-want +got):\n%s", diff)
	}
	// lib_b should also have been bumped to minor bump since lib_a changed, and its dependency updated
	if diff := cmp.Diff("0.5.0", cfg.Libraries[1].Version); diff != "" {
		t.Errorf("lib_b version mismatch (-want +got):\n%s", diff)
	}

	// Verify pubspecs
	verifyPubspec := func(path, expectedVersion, expectedDeps string) {
		content, err := os.ReadFile(filepath.Join(path, "pubspec.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if !strings.Contains(text, "version: "+expectedVersion) {
			t.Errorf("Pubspec at %s missing version %s", path, expectedVersion)
		}
		if expectedDeps != "" && !strings.Contains(text, expectedDeps) {
			t.Errorf("Pubspec at %s missing dependency %s", path, expectedDeps)
		}
	}

	verifyPubspec(outputA, "1.0.0", "")
	verifyPubspec(outputB, "0.5.0", "  lib_a: ^1.0.0")
}
