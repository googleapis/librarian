// Copyright 2025 Google LLC
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

package golang

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

const (
	envGoBin = "GOBIN"
	toolsDir = "go_tools"
)

var (
	// errMissingToolVersion indicates a go tool entry is missing its version.
	errMissingToolVersion = errors.New("go tool missing version")
	// errNoToolsSpecified indicates no Go tools were provided in the configuration.
	errNoToolsSpecified = errors.New("no tools specified in configuration")
)

// Install installs the tools required for Go library generation.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Go) == 0 {
		return errNoToolsSpecified
	}
	return installGoTools(ctx, tools.Go)
}

// InstallDir gets the directory where tools should be installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, toolsDir))
}

func installGoTools(ctx context.Context, goTools []*config.GoTool) error {
	installDir, err := InstallDir()
	if err != nil {
		return err
	}
	env := map[string]string{envGoBin: installDir}
	for _, tool := range goTools {
		if tool.Version == "" {
			return fmt.Errorf("%w: %s", errMissingToolVersion, tool.Name)
		}
		toolStr := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := runWithEnv(ctx, env, command.Go, "install", toolStr); err != nil {
			return err
		}
	}
	return nil
}
