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
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

const (
	goBinEnvVar        = "GOBIN"
	librarianBinEnvVar = "LIBRARIAN_INSTALL_DIR"
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

func installGoTools(ctx context.Context, goTools []*config.GoTool) error {
	installDir, err := getInstallDir()
	if err != nil {
		return err
	}
	// Go uses the GOBIN environment variable to determine where to install tools,
	// and it does not allow to specify the installation directory as a command-line argument.
	// We set it for the duration of the installation.
	originalGOBIN := os.Getenv(goBinEnvVar)
	defer os.Setenv(goBinEnvVar, originalGOBIN)
	env := map[string]string{goBinEnvVar: installDir}
	for _, tool := range goTools {
		if tool.Version == "" {
			return fmt.Errorf("%w: %s", errMissingToolVersion, tool.Name)
		}
		toolStr := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := command.RunWithEnv(ctx, env, command.Go, "install", toolStr); err != nil {
			return err
		}
	}
	return nil
}

func getInstallDir() (string, error) {
	dir := os.Getenv(librarianBinEnvVar)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dir = filepath.Join(home, "go_tools", "bin")
	}
	return filepath.Abs(dir)
}
