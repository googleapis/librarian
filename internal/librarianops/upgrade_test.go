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
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"golang.org/x/mod/semver"
)

func TestRunUpgrade(t *testing.T) {
	wantVersion, err := getLibrarianVersionAtMain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !semver.IsValid(wantVersion) {
		t.Fatalf("version from getLibrarianVersionAtMain %q is not a valid semantic version", wantVersion)
	}

	repoDir := t.TempDir()
	configPath := generateLibrarianConfigPath(t, repoDir)
	initialConfig := &config.Config{
		Language: "rust",
		Version:  "v0.1.0",
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatal(err)
	}

	gotVersion, err := runUpgrade(t.Context(), repoDir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantVersion, gotVersion); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotConfig, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}

	wantConfig := &config.Config{
		Language: "rust",
		Version:  wantVersion,
	}
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunUpgrade_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		setup     func(t *testing.T) (repoDir string)
		wantError string
	}{
		{
			name: "getLibrarianVersionAtMain error",
			setup: func(t *testing.T) string {
				// Make the "go" command fail by setting an invalid PATH.
				t.Setenv("PATH", t.TempDir())
				return t.TempDir()
			},
			wantError: "failed to get latest librarian version",
		},
		{
			name: "UpdateLibrarianVersion error",
			setup: func(t *testing.T) string {
				// Make writing the config file fail by creating a directory at its path.
				repoDir := t.TempDir()
				configPath := generateLibrarianConfigPath(t, repoDir)
				if err := os.Mkdir(configPath, 0755); err != nil {
					t.Fatal(err)
				}
				return repoDir
			},
			wantError: "failed to update librarian version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := test.setup(t)
			_, gotErr := runUpgrade(t.Context(), repoDir)
			if gotErr == nil {
				t.Fatal("got nil, want error")
			}
			// Error is dynamic so just checking the substring.
			if !strings.Contains(gotErr.Error(), test.wantError) {
				t.Errorf("error detail mismatch\ngot:  %q\nwant substring: %q", gotErr.Error(), test.wantError)
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	// Chdir is necessary because the upgrade command's -C flag defaults to the
	// current working directory.
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	configPath := generateLibrarianConfigPath(t, ".")
	initialConfig := &config.Config{
		Language: "rust",
		Version:  "v0.1.0",
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatal(err)
	}

	cmd := upgradeCommand()
	if err := cmd.Run(t.Context(), []string{"-C", "."}); err != nil {
		t.Error(err)
	}
}

func TestUpgradeCommand_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		setup     func(t *testing.T)
		wantError string
	}{
		{
			name:      "wrong arguments",
			args:      []string{},
			setup:     func(t *testing.T) {},
			wantError: "usage: librarianops <command> <repo> or librarianops <command> -C <dir>",
		},
		{
			name: "runUpgrade error",
			args: []string{"-C", "."},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantError: "failed to get latest librarian version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			test.setup(t)

			cmd := upgradeCommand()
			err := cmd.Run(t.Context(), test.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			// Error is dynamic so just checking the substring.
			if !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("error mismatch\ngot: %q, want substring: %q", err.Error(), test.wantError)
			}
		})
	}
}

func generateLibrarianConfigPath(t *testing.T, repoDir string) string {
	t.Helper()
	return filepath.Join(repoDir, "librarian.yaml")
}
