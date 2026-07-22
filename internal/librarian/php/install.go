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

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/tool/pip"
)

const (
	generatorVersion = "v1.21.2"
	generatorSHA256  = "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518"
	toolsDir         = "php_tools"
)

var (
	errMissingRepo = errors.New("repo URL missing")
)

// Install installs the PHP generator tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || (len(tools.Composer) == 0 && len(tools.Pip) == 0) {
		_, err := installGenerator(ctx)
		return err
	}

	bin, err := binDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bin, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	for _, tool := range tools.Composer {
		if tool.Repo == "" {
			return fmt.Errorf("%w: composer tool %s", errMissingRepo, tool.Name)
		}
		dir, err := fetch.Repo(ctx, tool.Repo, tool.Version, tool.SHA256)
		if err != nil {
			return fmt.Errorf("fetching %s: %w", tool.Name, err)
		}

		if err := command.RunInDir(ctx, dir, "composer", "install", "--no-dev", "--no-interaction", "--prefer-dist"); err != nil {
			return err
		}

		isGenerator := filepath.Base(tool.Repo) == "gapic-generator-php"

		var wrapperName, wrapperContent string
		if isGenerator {
			phpPath, err := exec.LookPath("php")
			if err != nil {
				return fmt.Errorf("failed to find php: %w", err)
			}
			destPath := filepath.Join(dir, "src", "Main.php")
			wrapperName = filepath.Base(tool.Repo)
			wrapperContent = fmt.Sprintf("#!/bin/bash\nexec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir \"$GOOGLEAPIS_DIR\" \"$@\"\n", phpPath, destPath)
		} else {
			destPath := filepath.Join(dir, "vendor", "bin", tool.Name)
			wrapperName = tool.Name
			wrapperContent = fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", destPath)
		}

		if err := createBinWrapper(wrapperName, wrapperContent, bin); err != nil {
			return err
		}
	}
	// The PHP client library generation process relies on Python-based
	// tools (such as synthtool or owlbot) for post-processing and generation.
	if len(tools.Pip) > 0 {
		if err := pip.Install(ctx, tools.Pip); err != nil {
			return err
		}
	}
	// Install PNPM tools
	if len(tools.PNPM) > 0 {
		if err := nodejs.InstallPNPM(ctx, tools.PNPM); err != nil {
			return fmt.Errorf("failed to install pnpm tools: %w", err)
		}
	}
	return nil
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

// createBinWrapper creates a shell wrapper script in the bin directory that forwards executions to the tool.
func createBinWrapper(wrapperName, wrapperContent, binDir string) error {
	wrapperPath := filepath.Join(binDir, wrapperName)
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for wrapper: %w", err)
	}
	_ = os.Remove(wrapperPath)
	f, err := os.OpenFile(wrapperPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(wrapperContent)
	return err
}
