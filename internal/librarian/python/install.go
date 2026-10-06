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

	"runtime"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
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
	if err := extractTemplates(); err != nil {
		return err
	}
	return installPandocBinary(ctx, tools.Pip)
}

// binDir gets the directory where Python tool executables are stored.
func binDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "bin"), nil
}

// pandocPath returns the deterministic path to the pandoc executable in the tools directory.
func pandocPath() (string, error) {
	bin, err := binDir()
	if err != nil {
		return "", err
	}
	pandoc := filepath.Join(bin, "pandoc")
	if runtime.GOOS == "windows" {
		pandoc += ".exe"
	}
	return pandoc, nil
}

// toolsEnv returns an environment map with the Python tools bin directory prepended to PATH
// and PYPANDOC_PANDOC set to the deterministic pandoc binary path if available.
func toolsEnv() (map[string]string, error) {
	bin, err := binDir()
	if err != nil {
		return nil, err
	}
	env := map[string]string{
		"PATH": bin,
	}
	if envVal := os.Getenv("PYPANDOC_PANDOC"); envVal != "" {
		env["PYPANDOC_PANDOC"] = envVal
		return env, nil
	}
	pandoc, err := pandocPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(pandoc); err == nil {
		env["PYPANDOC_PANDOC"] = pandoc
	}
	return env, nil
}

// installPandocBinary extracts the pandoc binary from pypandoc-binary into the tools bin directory.
func installPandocBinary(ctx context.Context, tools []*config.PipTool) error {
	var pypandocTool *config.PipTool
	for _, tool := range tools {
		if tool.Name == "pypandoc-binary" {
			pypandocTool = tool
			break
		}
	}
	if pypandocTool == nil {
		return nil
	}

	bin, err := binDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	dest, err := pandocPath()
	if err != nil {
		return err
	}

	versionFile := filepath.Join(bin, ".pypandoc-binary-version")
	if pypandocTool.Version != "" {
		if data, err := os.ReadFile(versionFile); err == nil && string(data) == pypandocTool.Version {
			if _, err := os.Stat(dest); err == nil {
				return nil
			}
		}
	}

	tmpDir, err := os.MkdirTemp("", "pypandoc-binary-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory for pypandoc-binary: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pkgSpec := pypandocTool.Name
	if pypandocTool.Version != "" {
		pkgSpec = fmt.Sprintf("%s==%s", pypandocTool.Name, pypandocTool.Version)
	}
	if pypandocTool.Package != "" {
		pkgSpec = pypandocTool.Package
	}
	if pypandocTool.LocalPath != "" {
		absPath, err := filepath.Abs(pypandocTool.LocalPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for %s: %w", pypandocTool.LocalPath, err)
		}
		pkgSpec = absPath
	}

	if err := command.Run(ctx, "pip", "install", "--no-deps", "--target", tmpDir, pkgSpec); err != nil {
		return fmt.Errorf("failed to extract pandoc binary from %s: %w", pypandocTool.Name, err)
	}

	binaryName := "pandoc"
	if runtime.GOOS == "windows" {
		binaryName = "pandoc.exe"
	}
	src := filepath.Join(tmpDir, "pypandoc", "files", binaryName)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("pandoc binary not found in %s: %w", pypandocTool.Name, err)
	}

	if err := filesystem.CopyFile(src, dest); err != nil {
		return fmt.Errorf("failed to copy pandoc binary: %w", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return fmt.Errorf("failed to make pandoc binary executable: %w", err)
	}
	if pypandocTool.Version != "" {
		_ = os.WriteFile(versionFile, []byte(pypandocTool.Version), 0o644)
	}
	return nil
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
