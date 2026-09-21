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

package testhelper

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/cache"
)

func TestFindManagedProtoc(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)

	if _, ok := findManagedProtoc(); ok {
		t.Fatal("expected false with empty bin directory")
	}

	protocBin := "protoc"
	if runtime.GOOS == "windows" {
		protocBin = "protoc.exe"
	}
	protocPath := filepath.Join(binDir, "protoc", "v33.2", "bin", protocBin)
	if err := os.MkdirAll(filepath.Dir(protocPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protocPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := findManagedProtoc()
	if !ok {
		t.Fatal("expected true with managed protoc binary present")
	}
	if got != protocPath {
		t.Fatalf("expected %q, got %q", protocPath, got)
	}
}

func TestFindManagedProtoc_MultipleVersions(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)

	protocBin := "protoc"
	if runtime.GOOS == "windows" {
		protocBin = "protoc.exe"
	}
	// Install multiple versions including one that lexicographically
	// sorts lower but is numerically higher (v33.10 vs v33.2), and a
	// prerelease (v26.0-rc1) that must rank below its stable release.
	for _, ver := range []string{"v25.1", "v33.2", "v26.0-rc1", "v26.0", "v33.10"} {
		p := filepath.Join(binDir, "protoc", ver, "bin", protocBin)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Also create a non-directory file and a directory without the "v"
	// prefix; both must be ignored.
	if err := os.WriteFile(filepath.Join(binDir, "protoc", "README"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(binDir, "protoc", "latest", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := findManagedProtoc()
	if !ok {
		t.Fatal("expected true with managed protoc binaries present")
	}
	want := filepath.Join(binDir, "protoc", "v33.10", "bin", protocBin)
	if got != want {
		t.Fatalf("expected newest version %q, got %q", want, got)
	}
}

func TestCompareProtocVersions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		newer string
		older string
	}{
		{"numeric minor", "v33.10", "v33.2"},
		{"patch comparison", "v3.20.3", "v3.20.2"},
		{"patch vs no patch", "v3.20.3", "v3.20"},
		{"stable beats prerelease", "v26.0", "v26.0-rc1"},
		{"prerelease ordering", "v26.0-rc2", "v26.0-rc1"},
		{"higher minor", "v26.0", "v25.99"},
		{"higher major", "v34.0", "v33.10"},
		{"three-part prerelease", "v3.20.3", "v3.20.3-rc1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			a, ok := parseProtocVersion(test.newer)
			if !ok {
				t.Fatalf("failed to parse %q", test.newer)
			}
			b, ok := parseProtocVersion(test.older)
			if !ok {
				t.Fatalf("failed to parse %q", test.older)
			}
			if c := compareProtocVersions(a, b); c <= 0 {
				t.Errorf("expected %s > %s, got compare = %d", test.newer, test.older, c)
			}
			if c := compareProtocVersions(b, a); c >= 0 {
				t.Errorf("expected %s < %s, got compare = %d", test.older, test.newer, c)
			}
		})
	}
}

func TestParseProtocVersion_Error(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"latest", "foo", "v", "vabc", "v33", "33.2", "v1.2.3.4"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, ok := parseProtocVersion(name); ok {
				t.Errorf("expected parseProtocVersion(%q) to fail", name)
			}
		})
	}
}

func TestRequireCommand_ManagedProtoc(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv(cache.EnvLibrarianBin, binDir)

	protocBin := "protoc"
	if runtime.GOOS == "windows" {
		protocBin = "protoc.exe"
	}
	protocPath := filepath.Join(binDir, "protoc", "v33.2", "bin", protocBin)
	if err := os.MkdirAll(filepath.Dir(protocPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protocPath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Clear PATH so exec.LookPath won't find protoc initially.
	t.Setenv("PATH", "")

	// RequireCommand should not skip because managed protoc exists,
	// and it should prepend the managed binary's directory to PATH.
	RequireCommand(t, "protoc")
	if t.Skipped() {
		t.Fatal("expected test not to be skipped when managed protoc exists")
	}

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Fatalf("expected managed protoc to be on PATH: %v", err)
	}
	if got := os.Getenv("PATH"); strings.HasSuffix(got, string(filepath.ListSeparator)) {
		t.Errorf("PATH has trailing separator: %q", got)
	}
}
