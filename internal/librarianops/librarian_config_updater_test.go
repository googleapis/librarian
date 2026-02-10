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
)

func TestGenerateLibrarianConfigPath(t *testing.T) {
	repoDir := "/path/to/repo"
	expectedPath := "/path/to/repo/librarian.yaml"
	generatedPath := GenerateLibrarianConfigPath(repoDir)
	if diff := cmp.Diff(expectedPath, generatedPath); diff != "" {
		t.Errorf("mismatch (-expected +generated):\n%s", diff)
	}
}

func TestGetConfigFile(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
	}{
		{
			name:    "Success",
			version: "v0.0.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "librarian.yaml")

			initialConfig := &config.Config{
				Version: test.version,
			}
			if err := yaml.Write(configPath, initialConfig); err != nil {
				t.Fatalf("failed to write initial config file: %v", err)
			}

			got, err := getConfigFile(configPath)
			if err != nil {
				t.Fatalf("getConfigFile() failed: %v", err)
			}
			want := initialConfig
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetConfigFile_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		setup      func(t *testing.T) string
		wantErrStr string
	}{
		{
			name: "File not found",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			wantErrStr: "config file does not exist at path",
		},
		{
			name: "Invalid YAML",
			setup: func(t *testing.T) string {
				configPath := filepath.Join(t.TempDir(), "librarian.yaml")
				if err := os.WriteFile(configPath, []byte("invalid-yaml: ["), 0644); err != nil {
					t.Fatalf("failed to write invalid config file: %v", err)
				}
				return configPath
			},
			wantErrStr: "did not find expected node content",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			configPath := test.setup(t)
			_, err := getConfigFile(configPath)
			if err == nil {
				t.Fatal("getConfigFile() expected an error, but got nil")
			}
			if !strings.Contains(err.Error(), test.wantErrStr) {
				t.Errorf("getConfigFile() error = %v, want substring %q", err, test.wantErrStr)
			}
		})
	}
}

func TestUpdateLibrarianVersionInConfigFile(t *testing.T) {
	for _, test := range []struct {
		name           string
		initialVersion string
		newVersion     string
	}{
		{
			name:           "Success",
			initialVersion: "v0.0.1",
			newVersion:     "v0.1.0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "librarian.yaml")

			initialConfig := &config.Config{
				Version: test.initialVersion,
			}
			if err := yaml.Write(configPath, initialConfig); err != nil {
				t.Fatalf("failed to write initial config file: %v", err)
			}

			if err := UpdateLibrarianVersionInConfigFile(test.newVersion, configPath); err != nil {
				t.Fatalf("UpdateLibrarianVersionInConfigFile() failed: %v", err)
			}

			updatedConfig, err := getConfigFile(configPath)
			if err != nil {
				t.Fatalf("getConfigFile() failed: %v", err)
			}

			wantConfig := initialConfig
			wantConfig.Version = test.newVersion

			if diff := cmp.Diff(wantConfig, updatedConfig); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateLibrarianVersionInConfigFile_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		newVersion string
		setup      func(t *testing.T) string
		wantErrStr string
	}{
		{
			name:       "File not found",
			newVersion: "v0.1.0",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			wantErrStr: "config file does not exist at path",
		},
		{
			name:       "Invalid YAML",
			newVersion: "v0.1.0",
			setup: func(t *testing.T) string {
				configPath := filepath.Join(t.TempDir(), "librarian.yaml")
				if err := os.WriteFile(configPath, []byte("invalid-yaml: ["), 0644); err != nil {
					t.Fatalf("failed to write invalid config file: %v", err)
				}
				return configPath
			},
			wantErrStr: "did not find expected node content",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			configPath := test.setup(t)
			err := UpdateLibrarianVersionInConfigFile(test.newVersion, configPath)
			if err == nil {
				t.Fatal("UpdateLibrarianVersionInConfigFile() expected an error, but got nil")
			}
			if !strings.Contains(err.Error(), test.wantErrStr) {
				t.Errorf("UpdateLibrarianVersionInConfigFile() error = %v, want substring %q", err, test.wantErrStr)
			}
		})
	}
}
