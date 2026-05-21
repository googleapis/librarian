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

// Package pip provides utility functions for installing Python packages via pip.
package pip

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// ErrInstall indicates a failure to install pip packages.
var ErrInstall = errors.New("failed to install python packages")

// ErrLocalPathNotFound indicates that the specified local package path does not exist.
var ErrLocalPathNotFound = errors.New("local pip package path not found")

// Install installs a list of pip tools into the environment.
func Install(ctx context.Context, tools []*config.PipTool) error {
	var installTargets []string
	for _, tool := range tools {
		if tool.LocalPath != "" {
			absPath, err := filepath.Abs(tool.LocalPath)
			if err != nil {
				return fmt.Errorf("failed to resolve absolute path for %s: %w", tool.LocalPath, err)
			}
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("%w: %w", ErrLocalPathNotFound, err)
			}
			installTargets = append(installTargets, absPath)
			continue
		}
		if tool.Package != "" {
			installTargets = append(installTargets, tool.Package)
			continue
		}
		if tool.Version != "" {
			installTargets = append(installTargets, fmt.Sprintf("%s==%s", tool.Name, tool.Version))
			continue
		}
		installTargets = append(installTargets, tool.Name)
	}
	args := []string{"install"}
	args = append(args, installTargets...)
	if err := command.RunStreaming(ctx, "pip", args...); err != nil {
		return fmt.Errorf("%w: %w", ErrInstall, err)
	}
	return nil
}
