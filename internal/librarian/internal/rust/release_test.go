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

	"github.com/google/go-cmp/cmp"
	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

const (
	storageDir      = "src/storage"
	storageCargo    = "src/storage/Cargo.toml"
	storageName     = "google-cloud-storage"
	storageInitial  = "1.0.0"
	storageReleased = "1.1.0"

	secretmanagerDir      = "src/secretmanager"
	secretmanagerCargo    = "src/secretmanager/Cargo.toml"
	secretmanagerName     = "google-cloud-secretmanager-v1"
	secretmanagerInitial  = "1.5.3"
	secretmanagerReleased = "1.6.0"
)

func TestReleaseAll(t *testing.T) {
	cfg := setupRelease(t)
	got, err := ReleaseAll(cfg)
	if err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, storageCargo, storageReleased)
	checkCargoVersion(t, secretmanagerCargo, secretmanagerReleased)
	checkLibraryVersion(t, got, storageName, storageReleased)
	checkLibraryVersion(t, got, secretmanagerName, secretmanagerReleased)
}

func TestReleaseOne(t *testing.T) {
	cfg := setupRelease(t)
	got, err := ReleaseLibrary(cfg, storageName)
	if err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, storageCargo, storageReleased)
	checkCargoVersion(t, secretmanagerCargo, secretmanagerInitial)
	checkLibraryVersion(t, got, storageName, storageReleased)
	checkLibraryVersion(t, got, secretmanagerName, secretmanagerInitial)
}

func setupRelease(t *testing.T) *config.Config {
	t.Helper()
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "taplo")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createCrate(t, storageDir, storageName, storageInitial)
	createCrate(t, secretmanagerDir, secretmanagerName, secretmanagerInitial)
	return &config.Config{
		Libraries: []*config.Library{
			{
				Name:    storageName,
				Version: storageInitial,
			},
			{
				Name:    secretmanagerName,
				Version: secretmanagerInitial,
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

func checkLibraryVersion(t *testing.T, cfg *config.Config, name, wantVersion string) {
	t.Helper()
	for _, lib := range cfg.Libraries {
		if lib.Name == name {
			if lib.Version != wantVersion {
				t.Errorf("library %q version mismatch: want %q, got %q", name, wantVersion, lib.Version)
			}
			return
		}
	}
	t.Errorf("library %q not found in config", name)
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		config      *config.Config
		want        *config.Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			want: &config.Library{Name: "example-library"},
		},
		{
			name:        "no_library_in_config",
			libraryName: "example-library",
			config:      &config.Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := libraryByName(test.config, test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("libraryByName(%q): %v", test.libraryName, err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
