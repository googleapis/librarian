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

package php

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
)

const envPath = "PATH"

// Format formats a generated PHP library.
func Format(ctx context.Context, library *config.Library) error {
	prettierPath, pluginPath, err := prettierToolPaths()
	if err != nil {
		return err
	}
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	// Run prettier '**/Client/*' --write --parser=php --single-quote --print-width=120 --plugin=<pluginPath>
	return command.RunInDirWithEnv(ctx, outdir, prettierEnv(prettierPath), prettierPath,
		"**/Client/*",
		"--write",
		"--parser=php",
		"--single-quote",
		"--print-width=120",
		"--plugin="+pluginPath,
	)
}

// prettierToolPaths resolves and validates the paths for prettier and its PHP plugin.
func prettierToolPaths() (string, string, error) {
	nodeInstallDir, err := nodejs.InstallDir()
	if err != nil {
		return "", "", err
	}
	prettierPath := filepath.Join(nodeInstallDir, "bin", "prettier")
	if _, err := os.Stat(prettierPath); err != nil {
		return "", "", fmt.Errorf("prettier not found at %s: %w", prettierPath, err)
	}
	// In pnpm 7.16+ and pnpm 8, global packages are installed in PNPM_HOME/global/5.
	// In librarian, PNPM_HOME is configured to nodeInstallDir/bin.
	// Note: pnpm 9+ uses hashed directories for global packages, which is not supported by this path resolution.
	pluginPath := filepath.Join(nodeInstallDir, "bin", "global", "5", "node_modules", "@prettier", "plugin-php")
	if _, err := os.Stat(pluginPath); err != nil {
		return "", "", fmt.Errorf("prettier PHP plugin not found at %s: %w", pluginPath, err)
	}

	return prettierPath, pluginPath, nil
}

// prettierEnv returns the environment variables required to run prettier.
func prettierEnv(prettierPath string) map[string]string {
	return map[string]string{
		envPath: filepath.Dir(prettierPath),
	}
}
