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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
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

func TestReleaseOne(t *testing.T) {
	cfg := setupRelease(t)
	err := ReleaseLibrary(cfg.Libraries[0], storageDir)
	if err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, storageCargo, storageReleased)
	checkCargoVersion(t, secretmanagerCargo, secretmanagerInitial)
	checkLibraryVersion(t, cfg.Libraries[0], storageReleased)
	checkLibraryVersion(t, cfg.Libraries[1], secretmanagerInitial)
}

func setupRelease(t *testing.T) *config.Config {
	t.Helper()
	testhelpers.RequireCommand(t, "cargo")
	testhelpers.RequireCommand(t, "taplo")
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

func TestDeriveSrcPath(t *testing.T) {
	for _, test := range []struct {
		name   string
		config *config.Config
		want   string
	}{
		{
			name: "use library output",
			config: &config.Config{
				Default: &config.Default{
					Output: "ignored",
				},
				Libraries: []*config.Library{
					{Output: "src/lib/dir"},
				},
			},
			want: "src/lib/dir",
		},
		{
			name: "use channel path",
			config: &config.Config{
				Default: &config.Default{
					Output: "src/",
				},
				Libraries: []*config.Library{{
					Channels: []*config.Channel{
						{Path: "channel/dir"},
					},
				},
				},
			},
			want: "src/channel/dir",
		},
		{
			name: "use library name",
			config: &config.Config{
				Default: &config.Default{
					Output: "src/",
				},
				Libraries: []*config.Library{{
					Name: "lib-name",
				},
				},
			},
			want: "src/lib/name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveSrcPath(test.config.Libraries[0], test.config)
			if got != test.want {
				t.Errorf("got derived source path  %s, wanted %s", got, test.want)
			}
		})
	}
}

func TestNoCargoFile(t *testing.T) {
	got := ReleaseLibrary(&config.Library{}, "")
	if got == nil {
		t.Errorf("Expected error reading cargo file but got %v", got)
	}
}
