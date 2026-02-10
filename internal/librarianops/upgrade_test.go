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

package librarianops

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"golang.org/x/mod/semver"
)

func TestRunUpgrade(t *testing.T) {
	wantVersion, err := GetLatestLibrarianVersion(t.Context())
	if err != nil {
		t.Fatalf("GetLatestLibrarianVersion() failed: %v", err)
	}
	if !semver.IsValid(wantVersion) {
		t.Fatalf("version from GetLatestLibrarianVersion %q is not a valid semantic version", wantVersion)
	}

	repoDir := t.TempDir()
	configPath := GenerateLibrarianConfigPath(repoDir)
	initialConfig := &config.Config{
		Language: "rust",
		Version:  "v0.1.0",
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatalf("Failed to write initial librarian.yaml: %v", err)
	}

	gotVersion, err := runUpgrade(t.Context(), repoDir)
	if err != nil {
		t.Fatalf("runUpgrade failed: %v", err)
	}

	if diff := cmp.Diff(wantVersion, gotVersion); diff != "" {
		t.Errorf("runUpgrade() version mismatch (-want +got):\n%s", diff)
	}

	gotConfig, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("Failed to read librarian.yaml: %v", err)
	}

	wantConfig := &config.Config{
		Language: "rust",
		Version:  wantVersion,
	}
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestRunUpgrade_Error(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T) (repoDir string)
	}{
		{
			name: "GetLatestLibrarianVersion error",
			setup: func(t *testing.T) string {
				// Make the "go" command fail by setting an invalid PATH.
				oldPath := os.Getenv("PATH")
				t.Setenv("PATH", t.TempDir())
				t.Cleanup(func() { t.Setenv("PATH", oldPath) })
				return t.TempDir()
			},
		},
		{
			name: "UpdateLibrarianVersion error",
			setup: func(t *testing.T) string {
				// Make writing the config file fail by creating a directory at its path.
				repoDir := t.TempDir()
				configPath := GenerateLibrarianConfigPath(repoDir)
				if err := os.Mkdir(configPath, 0755); err != nil {
					t.Fatalf("Failed to create directory at config path: %v", err)
				}
				return repoDir
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := test.setup(t)
			if _, err := runUpgrade(t.Context(), repoDir); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("os.Chdir(%q) failed: %v", wd, err)
		}
	})

	for _, test := range []struct {
		name    string
		args    []string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "success",
			args: []string{"-C", "."},
			setup: func(t *testing.T) {
				configPath := "librarian.yaml"
				initialConfig := &config.Config{
					Language: "rust",
					Version:  "v0.1.0",
				}
				if err := yaml.Write(configPath, initialConfig); err != nil {
					t.Fatalf("Failed to write initial librarian.yaml: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:    "no repo arg",
			args:    []string{},
			setup:   func(t *testing.T) {},
			wantErr: true,
		},
		{
			name: "runUpgrade error",
			args: []string{"-C", "."},
			setup: func(t *testing.T) {
				oldPath := os.Getenv("PATH")
				t.Setenv("PATH", t.TempDir())
				t.Cleanup(func() { t.Setenv("PATH", oldPath) })
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := t.TempDir()
			if err := os.Chdir(repoDir); err != nil {
				t.Fatalf("os.Chdir(%q) failed: %v", repoDir, err)
			}
			test.setup(t)

			cmd := upgradeCommand()
			err := cmd.Run(t.Context(), test.args)
			if (err != nil) != test.wantErr {
				t.Errorf("cmd.Run() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
