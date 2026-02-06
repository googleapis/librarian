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

package upgrade

import (
	"context"
	"os"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"golang.org/x/mod/semver"
)

func TestRunUpgrade(t *testing.T) {
	repoDir := t.TempDir()
	configPath := GenerateLibrarianConfigPath(repoDir)
	initialConfig := &config.Config{
		Language: "rust",
		Version:  "v0.1.0",
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatalf("Failed to write initial librarian.yaml: %v", err)
	}

	updatedVersion, err := runUpgrade(repoDir)
	if err != nil {
		t.Fatalf("runUpgrade failed: %v", err)
	}

	if !semver.IsValid(updatedVersion) {
		t.Errorf("updated version %q is not a valid semantic version", updatedVersion)
	}

	got, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatalf("Failed to read librarian.yaml: %v", err)
	}

	if got.Version == initialConfig.Version {
		t.Errorf("expected version to be updated, but it is still %q", got.Version)
	}

	if got.Version != updatedVersion {
		t.Errorf("expected version in config to be %q, but got %q", updatedVersion, got.Version)
	}
}

func TestRunUpgrade_GetLatestLibrarianVersionError(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir())
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	repoDir := t.TempDir()
	if _, err := runUpgrade(repoDir); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRunUpgrade_UpdateLibrarianVersionError(t *testing.T) {
	repoDir := t.TempDir()
	configPath := GenerateLibrarianConfigPath(repoDir)
	if err := os.Mkdir(configPath, 0755); err != nil {
		t.Fatalf("Failed to create directory at config path: %v", err)
	}

	if _, err := runUpgrade(repoDir); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUpgradeCommand(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(originalWD) })

	configPath := "librarian.yaml"
	initialConfig := &config.Config{
		Language: "rust",
		Version:  "v0.1.0",
	}
	if err := yaml.Write(configPath, initialConfig); err != nil {
		t.Fatalf("Failed to write initial librarian.yaml: %v", err)
	}

	cmd := upgradeCommand()
	if err := cmd.Run(context.Background(), []string{"-C", "."}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpgradeCommand_NoRepo(t *testing.T) {
	cmd := upgradeCommand()
	if err := cmd.Run(context.Background(), []string{}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUpgradeCommand_RunUpgradeError(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir())
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	repoDir := t.TempDir()
	cmd := upgradeCommand()
	if err := cmd.Run(context.Background(), []string{"-C", repoDir}); err == nil {
		t.Error("expected error, got nil")
	}
}
