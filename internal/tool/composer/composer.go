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

// Package composer provides utility functions for installing PHP packages via Composer.
package composer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

// ErrMissingRepo indicates that a repository URL is missing for a Composer tool.
var ErrMissingRepo = errors.New("repo URL missing")

// Install installs a list of Composer tools into the environment.
func Install(ctx context.Context, tools []*config.ComposerTool, phpPath, bin string) error {
	for _, tool := range tools {
		if tool.Repo == "" {
			return fmt.Errorf("%w: composer tool %s", ErrMissingRepo, tool.Name)
		}
		dir, err := fetch.Repo(ctx, tool.Repo, tool.Version, tool.SHA256)
		if err != nil {
			return fmt.Errorf("fetching %s: %w", tool.Name, err)
		}
		if err := command.RunInDir(ctx, dir, "composer", "install", "--no-interaction", "--prefer-dist"); err != nil {
			return fmt.Errorf("failed to run composer install: %w", err)
		}
		wrapperName := filepath.Base(tool.Repo)
		if wrapperName == "gapic-generator-php" {
			// Currently, this assumes the tool is the gapic-generator-php. This specific
			// wrapper logic will not work for generic Composer tools because:
			// 1. It hardcodes the executable entry point to "src/Main.php" (ignoring Composer's vendor/bin/ paths).
			// 2. It injects specific PHP configurations (e.g. memory_limit=1024M) required to prevent the generator from crashing.
			// See https://github.com/googleapis/gapic-generator-php/commit/685b419f2220e2d19c74e7f1464067f995cf1a95
			// 3. It automatically injects the "--side_loaded_root_dir" argument which other tools will not expect.
			// (this argument is to pass through relative paths for config files)
			// TODO(https://github.com/googleapis/librarian/issues/7000): Remove the --side_loaded_root_dir once we pass full paths to generator
			destPath := filepath.Join(dir, "src", "Main.php")
			wrapperContent := phpWrapperContent(phpPath, destPath)
			if err := createBinWrapper(wrapperName, wrapperContent, bin); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("tool installation for non-generator composer tools is not yet supported")
		}
	}
	return nil
}

// phpWrapperContent generates the bash script content for the PHP tool wrapper.
func phpWrapperContent(phpExecutable, entrypoint string) string {
	return fmt.Sprintf("#!/bin/bash\nexec %q -d display_errors=stderr -d memory_limit=1024M %q --side_loaded_root_dir \"$GOOGLEAPIS_DIR\" \"$@\"\n", phpExecutable, entrypoint)
}

// createBinWrapper creates a shell wrapper script in the bin directory that forwards executions to the tool.
func createBinWrapper(wrapperName, content, binDir string) error {
	wrapperPath := filepath.Join(binDir, wrapperName)
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for wrapper: %w", err)
	}
	_ = os.Remove(wrapperPath)
	if err := os.WriteFile(wrapperPath, []byte(content), 0o755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}
	return nil
}
