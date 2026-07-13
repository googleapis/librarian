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

package php

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
)

const toolsDir = "php_tools"

var (
	errNoToolsSpecified  = errors.New("no tools.composer field specified in configuration")
	errCannotExtractRepo = errors.New("cannot extract repo from package URL")
	errMissingExecutable = errors.New("is not installed or not in PATH, which is required for PHP tool installation")
	errMissingPackageURL = errors.New("has build steps but no package URL")
)

// Install installs PHP tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Composer) == 0 {
		return errNoToolsSpecified
	}

	for _, cmd := range []string{"composer", "php"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s %w: %w", cmd, errMissingExecutable, err)
		}
	}

	for _, tool := range tools.Composer {
		if err := installComposerToolFromSource(ctx, tool); err != nil {
			return err
		}
	}
	return nil
}

func installComposerToolFromSource(ctx context.Context, tool *config.ComposerTool) error {
	if tool.Package == "" {
		return fmt.Errorf("composer tool %s %w", tool.Name, errMissingPackageURL)
	}
	repo, err := repoFromPackageURL(tool.Package)
	if err != nil {
		return err
	}
	dir, err := fetch.Repo(ctx, repo, tool.Version, tool.SHA256)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", tool.Name, err)
	}

	for _, cmd := range tool.Build {
		if err := runPHPBuildCmd(ctx, dir, cmd); err != nil {
			return err
		}
	}

	// TODO(https://github.com/googleapis/librarian/issues/6630): Remove this post-processing step
	// once generate.go is refactored to not rely on wrapper.sh existing globally.
	if tool.Name == "google/gapic-generator-php" {
		if err := createLegacyWrapper(dir); err != nil {
			return err
		}
	}

	return nil
}

func createLegacyWrapper(generatorDir string) error {
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return fmt.Errorf("failed to find php: %w", err)
	}
	wrapperPath := filepath.Join(generatorDir, "wrapper.sh")
	wrapperContent := fmt.Sprintf(`#!/bin/bash
exec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir "$GOOGLEAPIS_DIR" "$@"
`, phpPath, filepath.Join(generatorDir, "src/Main.php"))

	// Create script atomically
	f, err := os.OpenFile(wrapperPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil // Wrapper already exists
		}
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}
	defer f.Close()

	if _, err := f.Write([]byte(wrapperContent)); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}
	return nil
}

func runPHPBuildCmd(ctx context.Context, dir string, cmdStr string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// repoFromPackageURL extracts the repository path (e.g.,
// "github.com/googleapis/gapic-generator-php") from a GitHub archive URL.
func repoFromPackageURL(packageURL string) (string, error) {
	parts := strings.SplitN(packageURL, "/archive/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("%w %q", errCannotExtractRepo, packageURL)
	}
	return strings.TrimPrefix(parts[0], "https://"), nil
}

// InstallDir gets the directory where tools should be installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	absDir, err := filepath.Abs(filepath.Join(dir, toolsDir))
	if err != nil {
		return "", fmt.Errorf("failed to get install directory: %w", err)
	}
	return absDir, nil
}

// generatorDir dynamically locates the gapic-generator-php directory from librarian.yaml.
// TODO(https://github.com/googleapis/librarian/issues/6630): remove after generate.go is refactored.
func generatorDir(ctx context.Context) (string, error) {
	cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		return "", err
	}
	if cfg.Tools == nil {
		return "", errors.New("no tools specified")
	}
	for _, tool := range cfg.Tools.Composer {
		if tool.Name == "google/gapic-generator-php" {
			repo, err := repoFromPackageURL(tool.Package)
			if err != nil {
				return "", err
			}
			return fetch.Repo(ctx, repo, tool.Version, tool.SHA256)
		}
	}
	return "", errors.New("google/gapic-generator-php not found in composer tools")
}
