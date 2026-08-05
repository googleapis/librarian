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

package nodejs

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/pnpm"
)

const (
	toolsDir = "nodejs_tools"
)

var (
	errNoToolsSpecified = errors.New("no tools.pnpm field specified in configuration")
)

// Install installs Node.js tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.PNPM) == 0 {
		return errNoToolsSpecified
	}
	binDir, err := getBinDir()
	if err != nil {
		return err
	}
	return pnpm.Install(ctx, tools.PNPM, binDir)
}

// InstallDir gets the directory where tools should be installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, toolsDir))
}

// getBinDir returns the directory where Node.js tool executables are stored.
func getBinDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "bin"), nil
}

// getToolsEnv returns an environment map with the Node.js tools bin directory prepended to PATH.
func getToolsEnv() (map[string]string, error) {
	binDir, err := getBinDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{"PATH": binDir}, nil
}
