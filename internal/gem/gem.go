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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var (
	// errInstall indicates a failure to install gem packages.
	errInstall    = errors.New("failed to install ruby gems")
	errInvalidGem = errors.New("invalid gem tool")
)

// Install installs a list of gem tools into the environment.
func Install(ctx context.Context, tools []*config.GemTool) error {
	for _, tool := range tools {
		if tool == nil || tool.Name == "" || tool.Version == "" {
			return fmt.Errorf("%w: missing name or version", errInvalidGem)
		}
		args := []string{"install", tool.Name, "-v", tool.Version}
		if err := command.RunStreaming(ctx, "gem", args...); err != nil {
			return fmt.Errorf("%w: %w", errInstall, err)
		}
	}
	return nil
}
