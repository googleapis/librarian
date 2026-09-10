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
}
