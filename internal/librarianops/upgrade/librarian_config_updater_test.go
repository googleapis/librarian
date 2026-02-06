// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package upgrade

import (
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestGenerateLibrarianConfigPath(t *testing.T) {
	repoDir := "/path/to/repo"
	expectedPath := "/path/to/repo/librarian.yaml"
	actualPath := GenerateLibrarianConfigPath(repoDir)
	if actualPath != expectedPath {
		t.Errorf("generateLibrarianConfigPath() = %s; want %s", actualPath, expectedPath)
	}
}

func TestGetConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "librarian.yaml")
	initialVersion := "v0.0.1"

	initialConfig := &config.Config{
		Version: initialVersion,
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatalf("failed to write initial config file: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		cfg, err := getConfigFile(configPath)
		if err != nil {
			t.Fatalf("getConfigFile() failed: %v", err)
		}
		if cfg.Version != initialVersion {
			t.Errorf("wrong version, got: %s, want: %s", cfg.Version, initialVersion)
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.yaml")
		_, err := getConfigFile(nonExistentPath)
		if err == nil {
			t.Error("expected an error for a non-existent file, but got nil")
		}
	})
}

func TestUpdateLibrarianVersion(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "librarian.yaml")
	initialVersion := "v0.0.1"
	newVersion := "v0.1.0"

	initialConfig := &config.Config{
		Version: initialVersion,
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatalf("failed to write initial config file: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		if err := UpdateLibrarianVersion(newVersion, configPath); err != nil {
			t.Fatalf("updateLibrarianVersion() failed: %v", err)
		}

		updatedConfig, err := getConfigFile(configPath)
		if err != nil {
			t.Fatalf("getConfigFile() failed: %v", err)
		}

		if updatedConfig.Version != newVersion {
			t.Errorf("version not updated, got: %s, want: %s", updatedConfig.Version, newVersion)
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.yaml")
		err := UpdateLibrarianVersion(newVersion, nonExistentPath)
		if err == nil {
			t.Error("expected an error for a non-existent file, but got nil")
		}
	})
}
