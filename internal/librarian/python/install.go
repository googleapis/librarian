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

package python

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/pip"
)

//go:embed all:templates
var templatesFS embed.FS

const (
	toolsDir  = "python_tools"
	templates = "templates"
)

var (
	// ErrNoToolsSpecified indicates no pip tools were provided in the configuration.
	ErrNoToolsSpecified = errors.New("no tools.pip field specified in configuration")
)

// Install installs Python pip tool dependencies and extracts templates.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Pip) == 0 {
		return ErrNoToolsSpecified
	}
	if err := pip.Install(ctx, tools.Pip); err != nil {
		return err
	}
	return extractTemplates()
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

// extractTemplates extracts embedded templates into the tools directory.
func extractTemplates() error {
	dest, err := templateDirectory()
	if err != nil {
		return err
	}
	sub, err := fs.Sub(templatesFS, templates)
	if err != nil {
		return fmt.Errorf("failed to get templates sub-filesystem: %w", err)
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to clean templates directory: %w", err)
	}
	if err := os.CopyFS(dest, sub); err != nil {
		return fmt.Errorf("failed to extract templates: %w", err)
	}
	return nil
}

// templateDirectory gets the directory where templates are stored.
func templateDirectory() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, templates), nil
}
