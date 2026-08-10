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
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

var (
	// ErrMissingRepo indicates that a repository URL is missing for a Composer tool.
	ErrMissingRepo = errors.New("repo URL missing")

	// ErrInvalidTool indicates that a Composer tool configuration is invalid.
	ErrInvalidTool = errors.New("invalid tool configuration")
)

// Install installs a list of Composer tools into the environment.
// It also installs dependencies for the PHP project if a local_path tool (like "dev") is provided.
func Install(ctx context.Context, tools []*config.ComposerTool, phpPath, bin string) error {
	if err := verify(tools); err != nil {
		return err
	}
	for _, tool := range tools {
		var dir string
		var err error
		if tool.LocalPath != "" {
			dir, err = localPath(tool.LocalPath)
		} else {
			dir, err = fetch.Repo(ctx, tool.Repo, tool.Version, tool.SHA256)
			if err != nil {
				err = fmt.Errorf("fetching %s: %w", tool.Name, err)
			}
		}
		if err != nil {
			return err
		}
		if err := command.RunInDir(ctx, dir, "composer", "install", "--no-interaction", "--prefer-dist"); err != nil {
			return fmt.Errorf("failed to run composer install: %w", err)
		}
		if tool.Entrypoint == "" {
			continue // No wrapper needed
		}
		wrapperName := filepath.Base(tool.Name)
		destPath := filepath.Join(dir, tool.Entrypoint)
		wrapperContent := phpWrapperContent(phpPath, destPath)
		if err := createBinWrapper(wrapperName, wrapperContent, bin); err != nil {
			return err
		}
	}
	return nil
}

// localPath resolves and validates the absolute path for a local composer tool.
func localPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %s: %w", path, err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("failed to stat local composer path: %w", err)
	}
	return absPath, nil
}

// phpWrapperContent generates the bash script content for the PHP tool wrapper.
func phpWrapperContent(phpExecutable, entrypoint string) string {
	return fmt.Sprintf("#!/bin/bash\nexec %q -d display_errors=stderr -d memory_limit=1024M %q \"$@\"\n", phpExecutable, entrypoint)
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

func verify(tools []*config.ComposerTool) error {
	for _, tool := range tools {
		if tool.Name == "" {
			return fmt.Errorf("%w: name must be specified: %+v", ErrInvalidTool, tool)
		}
		if filepath.IsAbs(tool.Entrypoint) || strings.Contains(tool.Entrypoint, "..") {
			return fmt.Errorf("%w: entrypoint must be a clean relative path: %+v", ErrInvalidTool, tool)
		}
		hasLocal := tool.LocalPath != ""
		hasRemote := tool.Version != "" || tool.Repo != "" || tool.SHA256 != ""
		if hasLocal && hasRemote {
			return fmt.Errorf("%w: cannot specify both local_path and version/repo/sha256: %+v", ErrInvalidTool, tool)
		}
		if !hasLocal && !hasRemote {
			return fmt.Errorf("%w: must specify either local_path or version/repo/sha256: %+v", ErrInvalidTool, tool)
		}
		if hasRemote {
			if tool.Version == "" {
				return fmt.Errorf("%w: version must be specified: %+v", ErrInvalidTool, tool)
			}
			if tool.Repo == "" {
				return fmt.Errorf("%w: composer tool %s", ErrMissingRepo, tool.Name)
			}
			if tool.SHA256 == "" {
				return fmt.Errorf("%w: sha256 must be specified: %+v", ErrInvalidTool, tool)
			}
		}
	}
	return nil
}
