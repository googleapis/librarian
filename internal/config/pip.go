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
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
)

// ErrPipInstall indicates a failure to install pip packages.
var ErrPipInstall = errors.New("failed to install python packages")

// InstallPipTools installs a list of pip tools into the environment.
func InstallPipTools(ctx context.Context, tools []*PipTool) error {
	var installTargets []string
	for _, tool := range tools {
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
		return fmt.Errorf("%w: %w", ErrPipInstall, err)
	}
	return nil
}
