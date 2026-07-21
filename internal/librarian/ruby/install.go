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

package ruby

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"context"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/gem"
)

const toolsDir = "ruby_tools"

var (
	errNoGems            = errors.New("no gem tools specified")
	errMissingExecutable = errors.New("is not installed or not in PATH, which is required for Ruby tool installation")
)

// Install installs Ruby gem dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if err := verify(tools); err != nil {
		return err
	}
	installDir, err := InstallDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(installDir, "bin")
	return gem.Install(ctx, tools.Gem, binDir, installDir)
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

func verify(tools *config.Tools) error {
	if tools == nil || len(tools.Gem) == 0 {
		return errNoGems
	}

	for _, cmd := range []string{"gem"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s %w: %w", cmd, errMissingExecutable, err)
		}
	}
	return nil
}
