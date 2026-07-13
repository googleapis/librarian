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

// Package gem provides utility functions for installing Ruby gems via gem.
package gem

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var (
	// errInstall indicates a failure to install gem packages.
	errInstall = errors.New("failed to install ruby gems")
	// errInvalidGem indicates an invalid gem tool.
	errInvalidGem = errors.New("invalid gem tool")
	// errInvalidBinDir indicates an invalid bin directory.
	errInvalidBinDir = errors.New("bin dir is not an absolute path")
	// errInvalidInstallDir indicates an invalid install directory.
	errInvalidInstallDir = errors.New("install dir is not an absolute path")
)

// Install installs a list of gem tools into the environment.
func Install(ctx context.Context, tools []*config.GemTool, binDir, installDir string) error {
	if err := verify(tools, binDir, installDir); err != nil {
		return err
	}
	for _, tool := range tools {
		args := []string{
			"install",
			tool.Name,
			"-v", tool.Version,
			"--bindir", binDir,
			"--install-dir", installDir,
			// Skip the generation of the local documentation to make installation faster
			// and use less disk space.
			"--no-document",
		}
		if err := command.RunStreaming(ctx, "gem", args...); err != nil {
			return fmt.Errorf("%w: %w", errInstall, err)
		}
	}
	return nil
}

func verify(tools []*config.GemTool, binDir, installDir string) error {
	if !filepath.IsAbs(binDir) {
		return fmt.Errorf("%w: Ruby bin directory %q is not an absolute path", errInvalidBinDir, binDir)
	}
	if !filepath.IsAbs(installDir) {
		return fmt.Errorf("%w: Ruby install directory %q is not an absolute path", errInvalidInstallDir, installDir)
	}
	for _, tool := range tools {
		if tool.Name == "" || tool.Version == "" {
			return fmt.Errorf("%w: name and version must be specified: %+v", errInvalidGem, tool)
		}

	}
	return nil
}
