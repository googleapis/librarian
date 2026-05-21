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

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
)

// InstallPipTools installs a list of pip tools into the environment.
func InstallPipTools(ctx context.Context, tools []*PipTool) error {
	var installTargets []string
	hasLocal := false
	for _, tool := range tools {
		if tool.LocalPath != "" {
			absPath, err := filepath.Abs(tool.LocalPath)
			if err != nil {
				return fmt.Errorf("failed to resolve absolute path for local pip package %s: %w", tool.Name, err)
			}
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("local pip package path not found: %s: %w", absPath, err)
			}
			installTargets = append(installTargets, absPath)
			hasLocal = true
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
	if hasLocal {
		args = append(args, "--no-build-isolation")
	}
	args = append(args, installTargets...)

	if err := command.RunStreaming(ctx, "pip", args...); err != nil {
		return fmt.Errorf("failed to install python packages: %w", err)
	}
	return nil
}
