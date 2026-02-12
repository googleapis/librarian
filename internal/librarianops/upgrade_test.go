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
	"github.com/googleapis/librarian/internal/sample"
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
	t.Chdir(repoDir)
	configPath := generateLibrarianConfigPath(t, repoDir)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	initialConfig := sample.Config()
	initialConfig.Language = "fake"
	initialConfig.Version = "v0.1.0"
	initialConfig.Sources.Googleapis.Dir = googleapisDir
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

	wantConfig := initialConfig
	wantConfig.Version = wantVersion
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunUpgrade_Error(t *testing.T) {
	for _, test := range []struct {
		name           string
		setup          func(t *testing.T) (repoDir string)
		wantErrMessage string
	}{
		{
			name: "getLibrarianVersionAtMain error",
			setup: func(t *testing.T) string {
				// Make the "go" command fail by setting an invalid PATH.
				t.Setenv("PATH", t.TempDir())
				return t.TempDir()
			},
			wantErrMessage: "failed to get latest librarian version",
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
			wantErrMessage: "failed to update librarian version",
		},
		{
			name: "runLibrarianWithVersion error",
			setup: func(t *testing.T) string {
				repoDir := t.TempDir()
				configPath := generateLibrarianConfigPath(t, repoDir)
				// Use an invalid config that will cause `librarian generate` to fail.
				// An empty language should be invalid.
				cfg := &config.Config{Language: ""}
				if err := yaml.Write(configPath, cfg); err != nil {
					t.Fatal(err)
				}
				return repoDir
			},
			wantErrMessage: "failed to run librarian generate",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoDir := test.setup(t)
			_, gotErr := runUpgrade(t.Context(), repoDir)
			if gotErr == nil {
				t.Fatal("got nil, want error")
			}
			if !strings.Contains(gotErr.Error(), test.wantErrMessage) {
				t.Errorf("error detail mismatch\ngot:  %q\nwant substring: %q", gotErr.Error(), test.wantErrMessage)
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
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	initialConfig := sample.Config()
	initialConfig.Language = "fake"
	initialConfig.Version = "v0.1.0"
	initialConfig.Sources.Googleapis.Dir = googleapisDir
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
		name           string
		args           []string
		setup          func(t *testing.T)
		wantErrMessage string
	}{{
		name:           "usage error",
		args:           []string{},
		setup:          func(t *testing.T) {},
		wantErrMessage: "usage:",
	},
		{
			name: "runUpgrade error",
			args: []string{"-C", "."},
			setup: func(t *testing.T) {
				t.Setenv("PATH", t.TempDir())
			},
			wantErrMessage: "failed to get latest librarian version",
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
			if !strings.Contains(err.Error(), test.wantErrMessage) {
				t.Errorf("error mismatch\ngot: %q, want substring: %q", err.Error(), test.wantErrMessage)
			}
		})
	}
}

func generateLibrarianConfigPath(t *testing.T, repoDir string) string {
	t.Helper()
	return filepath.Join(repoDir, "librarian.yaml")
}
