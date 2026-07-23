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
	toolsDir = "php_tools"
)

var (
	errMissingRepo = errors.New("repo URL missing")
	errNoComposer  = errors.New("no composer tools specified")
)

// Install installs the PHP generator tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Composer) == 0 {
		return errNoComposer
	}

	phpPath, err := checkRequiredCommands()
	if err != nil {
		return err
	}

	bin, err := binDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
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
		if err := command.RunInDir(ctx, dir, "composer", "install", "--no-interaction", "--prefer-dist"); err != nil {
			return fmt.Errorf("failed to run composer install: %w", err)
		}

		// We will likely need to add branching logic here when we add other tools
		// (potentially the upcoming google-cloud tool in issue #6978).
		// Currently, this assumes the tool is the gapic-generator-php. This specific
		// wrapper logic will not work for generic Composer tools because:
		// 1. It hardcodes the executable entry point to "src/Main.php" (ignoring Composer's vendor/bin/ paths).
		// 2. It injects specific PHP configurations (e.g. memory_limit=1024M) required to prevent the generator from crashing.
		// 3. It automatically injects the "--side_loaded_root_dir" argument which other tools will not expect.
		// (this argument is to pass through relative paths for config files)
		// TODO(https://github.com/googleapis/librarian/issues/7000): Remove the --side_loaded_root_dir once we pass full paths to generator
		destPath := filepath.Join(dir, "src", "Main.php")
		wrapperName := filepath.Base(tool.Repo)
		wrapperContent := phpWrapperContent(phpPath, destPath)

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

func checkRequiredCommands() (string, error) {
	if _, err := exec.LookPath("composer"); err != nil {
		return "", fmt.Errorf("failed to find composer: %w", err)
	}
	phpPath, err := exec.LookPath("php")
	if err != nil {
		return "", fmt.Errorf("failed to find php: %w", err)
	}
	return phpPath, nil
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

// phpWrapperContent generates the bash script content for the PHP tool wrapper.
func phpWrapperContent(phpExecutable, entrypoint string) string {
	return fmt.Sprintf("#!/bin/bash\nexec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir \"$GOOGLEAPIS_DIR\" \"$@\"\n", phpExecutable, entrypoint)
}

// createBinWrapper creates a shell wrapper script in the bin directory that forwards executions to the tool.
func createBinWrapper(wrapperName, content, binDir string) (err error) {
	wrapperPath := filepath.Join(binDir, wrapperName)
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for wrapper: %w", err)
	}
	_ = os.Remove(wrapperPath)
	f, err := os.OpenFile(wrapperPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close wrapper script: %w", closeErr)
		}
	}()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}
	return nil
}
