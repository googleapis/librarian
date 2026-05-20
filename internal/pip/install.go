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

// Package pip provides functions for installing python packages using the pip command.
package pip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Install installs a list of pip tools.
func Install(ctx context.Context, tools []*config.PipTool) error {
	var installTargets []string
	hasLocal := false
	for _, t := range tools {
		if t.LocalPath != "" {
			absPath, err := filepath.Abs(t.LocalPath)
			if err != nil {
				return fmt.Errorf("failed to resolve absolute path for local pip package %s: %w", t.Name, err)
			}
			// Verify local path exists
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("local pip package path not found: %s: %w", absPath, err)
			}
			installTargets = append(installTargets, absPath)
			hasLocal = true
		} else if t.Package != "" {
			installTargets = append(installTargets, t.Package)
		} else if t.Version != "" {
			installTargets = append(installTargets, fmt.Sprintf("%s==%s", t.Name, t.Version))
		} else {
			installTargets = append(installTargets, t.Name)
		}
	}

	args := []string{"install"}
	if hasLocal {
		args = append(args, "--no-build-isolation")
	}
	args = append(args, installTargets...)

	fmt.Printf("Installing Python packages via pip...\n")
	if err := command.RunStreaming(ctx, "pip", args...); err != nil {
		return fmt.Errorf("failed to install python packages: %w", err)
	}

	return nil
}
