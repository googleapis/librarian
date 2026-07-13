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
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

const (
	generatorVersion = "v1.21.2"
	generatorSHA256  = "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518"
	toolsDir         = "php_tools"
)

var (
	errCannotExtractRepo = errors.New("cannot extract repo from package URL")
	errMissingPackageURL = errors.New("has build steps but no package URL")
)

// Install installs the PHP generator tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools != nil && len(tools.Composer) > 0 {
		for _, tool := range tools.Composer {
			if tool.Package == "" {
<<<<<<< HEAD
				return fmt.Errorf("composer tool %s %w", tool.Name, errMissingPackageURL)
=======
				return fmt.Errorf("%w: composer tool %s", errMissingPackageURL, tool.Name)
>>>>>>> e14a92a9 (feat(php): support dynamic composer installation)
			}
			repo, err := repoFromPackageURL(tool.Package)
			if err != nil {
				return err
			}
			dir, err := fetch.Repo(ctx, repo, tool.Version, tool.SHA256)
			if err != nil {
				return fmt.Errorf("fetching %s: %w", tool.Name, err)
			}

			for _, cmdStr := range tool.Build {
				if err := command.RunInDir(ctx, dir, "sh", "-c", cmdStr); err != nil {
					return err
				}
			}
		}
		return nil
	}
	_, err := installGenerator(ctx)
	return err
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

// binDir gets the directory where PHP tool executables are stored.
func binDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "bin"), nil
}

// installGenerator is a temp function for testing purposes.
// TODO(https://github.com/googleapis/librarian/issues/6630): remove after install command is ready.
func installGenerator(ctx context.Context) (string, error) {
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return "", fmt.Errorf("failed to find php: %w", err)
	}

	generatorDir, err := generatorDir(ctx)
	if err != nil {
		return "", err
	}
	// Ensure Composer dependencies are installed
	vendorDir := filepath.Join(generatorDir, "vendor")
	if _, err := os.Stat(vendorDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if _, err := exec.LookPath("composer"); err == nil {
			if err := command.RunInDir(ctx, generatorDir, "composer", "install"); err != nil {
				return "", fmt.Errorf("failed to run composer install: %w", err)
			}
		} else {
			composerPhar := filepath.Join(generatorDir, "rules_php_gapic", "resources", "composer.phar")
			if _, err := os.Stat(composerPhar); err == nil {
				if err := command.RunInDir(ctx, generatorDir, phpPath, composerPhar, "install"); err != nil {
					return "", fmt.Errorf("failed to run composer.phar install: %w", err)
				}
			} else {
				return "", fmt.Errorf("neither system composer nor composer.phar was found")
			}
		}
	}
	// Write wrapper script
	wrapperPath := filepath.Join(generatorDir, "wrapper.sh")
	wrapperContent := fmt.Sprintf(`#!/bin/bash
exec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir "$GOOGLEAPIS_DIR" "$@"
`, phpPath, filepath.Join(generatorDir, "src/Main.php"))

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return generatorDir, nil
}

func generatorDir(ctx context.Context) (string, error) {
	return fetch.Repo(ctx, "github.com/googleapis/gapic-generator-php", generatorVersion, generatorSHA256)
}
