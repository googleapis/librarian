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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const (
	storageDir      = "src/storage"
	storageCargo    = "src/storage/Cargo.toml"
	storageName     = "google-cloud-storage"
	storageInitial  = "1.0.0"
	storageReleased = "1.1.0"

	secretmanagerDir     = "src/secretmanager"
	secretmanagerCargo   = "src/secretmanager/Cargo.toml"
	secretmanagerName    = "google-cloud-secretmanager-v1"
	secretmanagerInitial = "1.5.3"
)

func TestBumpOne(t *testing.T) {
	cfg := setupRelease(t)
	if err := Bump(cfg.Libraries[0], storageReleased); err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, storageCargo, storageReleased)
	checkCargoVersion(t, secretmanagerCargo, secretmanagerInitial)
	checkLibraryVersion(t, cfg.Libraries[0], storageReleased)
	checkLibraryVersion(t, cfg.Libraries[1], secretmanagerInitial)
}

func setupRelease(t *testing.T) *config.Config {
	t.Helper()
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "taplo")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createCrate(t, storageDir, storageName, storageInitial)
	createCrate(t, secretmanagerDir, secretmanagerName, secretmanagerInitial)
	return &config.Config{
		Libraries: []*config.Library{
			{
				Name:    storageName,
				Version: storageInitial,
				Output:  storageDir,
			},
			{
				Name:    secretmanagerName,
				Version: secretmanagerInitial,
				Output:  secretmanagerDir,
			},
		},
	}
}

func createCrate(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, name, version)

	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatal(err)
	}
}

func checkCargoVersion(t *testing.T, path, wantVersion string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantLine := fmt.Sprintf(`version                = "%s"`, wantVersion)
	got := string(contents)
	if !strings.Contains(got, wantLine) {
		t.Errorf("%s version mismatch:\nwant line: %q\ngot:\n%s", path, wantLine, got)
	}
}

func checkLibraryVersion(t *testing.T, library *config.Library, wantVersion string) {
	t.Helper()
	if library.Version != wantVersion {
		t.Errorf("library %q version mismatch: want %q, got %q", library.Name, wantVersion, library.Version)
	}
}

func TestNoCargoFile(t *testing.T) {
	err := Bump(&config.Library{Version: "1.0.0", Output: "nonexistent/path"}, storageReleased)
	if err == nil {
		t.Error("expected error when Cargo.toml doesn't exist")
	}
}

func TestMissingVersion(t *testing.T) {
	err := Bump(&config.Library{}, "")
	if !errors.Is(err, errMissingVersion) {
		t.Errorf("expected error %v, got %v", errMissingVersion, err)
	}
}

func TestBumpLibraryNoVersion(t *testing.T) {
	testhelper.RequireCommand(t, "cargo")
	testhelper.RequireCommand(t, "taplo")

	const (
		libDir  = "src/test-lib"
		libName = "test-library"
	)
	for _, test := range []struct {
		name        string
		createCargo bool
		cargoVer    string
		wantVersion string
	}{
		{
			name:        "library.Version empty, Cargo.toml exists with 0.5.0, uses default 0.1.0 without bumping",
			createCargo: true,
			cargoVer:    "0.5.0",
			wantVersion: "0.1.0",
		},
		{
			name:        "library.Version empty, no Cargo.toml, uses default 0.1.0 without bumping",
			createCargo: false,
			wantVersion: "0.1.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			if err := os.MkdirAll(libDir, 0755); err != nil {
				t.Fatal(err)
			}
			if test.createCargo {
				createCrate(t, libDir, libName, test.cargoVer)
			}

			lib := &config.Library{
				Name:   libName,
				Output: libDir,
			}
			if err := Bump(lib, test.wantVersion); err != nil {
				t.Fatal(err)
			}
			checkLibraryVersion(t, lib, test.wantVersion)
			checkCargoVersion(t, filepath.Join(libDir, "Cargo.toml"), test.wantVersion)
		})
	}
}
