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

package swift

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageDirectory(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		if got := PackageDirectory(""); got != "" {
			t.Errorf("PackageDirectory(\"\") = %q, want \"\"", got)
		}
	})

	t.Run("no package.swift returns input dir", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "a", "b", "c")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if got := PackageDirectory(subDir); got != subDir {
			t.Errorf("PackageDirectory(%q) = %q, want %q", subDir, got, subDir)
		}
	})

	t.Run("at package root", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tempDir, "Package.swift"), []byte("// package\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := PackageDirectory(tempDir); got != tempDir {
			t.Errorf("PackageDirectory(%q) = %q, want %q", tempDir, got, tempDir)
		}
	})

	t.Run("nested subdirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		pkgRoot := filepath.Join(tempDir, "pkgs", "swift-test")
		nestedDir := filepath.Join(pkgRoot, "Sources", "TestModule", "generated")
		if err := os.MkdirAll(nestedDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgRoot, "Package.swift"), []byte("// package\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := PackageDirectory(nestedDir); got != pkgRoot {
			t.Errorf("PackageDirectory(%q) = %q, want %q", nestedDir, got, pkgRoot)
		}
	})

	t.Run("relative path ignores monorepo root Package.swift", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Chdir(tempDir)
		if err := os.WriteFile("Package.swift", []byte("// root package\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		relDir := filepath.Join("generated", "swift-google-rpc")
		if got := PackageDirectory(relDir); got != relDir {
			t.Errorf("PackageDirectory(%q) = %q, want %q", relDir, got, relDir)
		}
	})

	t.Run("fallback strips /Sources/ when Package.swift is absent", func(t *testing.T) {
		in := filepath.Join("pkgs", "swift-google-auth", "Sources", "GoogleAuth", "generated")
		want := filepath.Join("pkgs", "swift-google-auth")
		if got := PackageDirectory(in); got != want {
			t.Errorf("PackageDirectory(%q) = %q, want %q", in, got, want)
		}
		if got := PackageDirectory("Sources/MyTarget"); got != "." {
			t.Errorf("PackageDirectory(\"Sources/MyTarget\") = %q, want \".\"", got)
		}
	})
}
