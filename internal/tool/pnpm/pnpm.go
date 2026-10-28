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

// Package pnpm provides utilities to install pnpm packages and tools.
package pnpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

var (
	// ErrInstall indicates a failure to install pnpm packages or execute build steps.
	ErrInstall = errors.New("failed to install pnpm packages")

	// ErrCannotExtractRepo indicates that the repository path cannot be extracted from a package URL.
	ErrCannotExtractRepo = errors.New("cannot extract repo from package URL")

	// ErrMissingPackageURL indicates that a tool configuration has build steps but no package URL.
	ErrMissingPackageURL = errors.New("has build steps but no package URL")

	// ErrMissingSrcDir indicates that a tool configuration has build steps but no source directory.
	ErrMissingSrcDir = errors.New("has build steps but no source directory")

	// ErrMissingExecutable indicates that node or pnpm is not installed or not in PATH.
	ErrMissingExecutable = errors.New("missing required executable for pnpm tool installation")
)

// Install installs PNPM tools into the specified binDir environment.
func Install(ctx context.Context, pnpmTools []*config.PNPMTool, binDir string) error {
	if len(pnpmTools) == 0 {
		return nil
	}

	for _, cmd := range []string{"node", "pnpm"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%w: %s (%v)", ErrMissingExecutable, cmd, err)
		}
	}

	toolEnv, err := env(binDir)
	if err != nil {
		return err
	}

	for _, tool := range pnpmTools {
		if len(tool.Build) > 0 {
			if err := installFromSource(ctx, toolEnv, tool); err != nil {
				return err
			}
			continue
		}

		pkg := tool.Package
		if pkg == "" {
			pkg = fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		}
		if err := run(ctx, "", toolEnv, "add", "-g", pkg); err != nil {
			return fmt.Errorf("%w: %w", ErrInstall, err)
		}
	}
	return nil
}

// env constructs a transient environment variable block to configure pnpm.
//
// This redirects all globally-installed pnpm binaries to LIBRARIAN_BIN, and
// virtual stores / content-addressable storage caches to LIBRARIAN_CACHE.
// This enables complete environment caching and restore on CI runners,
// while permanently avoiding persistent side-effects on the host machine
// (it does not modify the user's personal ~/.config/pnpm/rc files).
func env(binDir string) ([]string, error) {
	cacheDir, err := cache.Directory()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve librarian cache directory: %w", err)
	}
	globalDir := filepath.Join(cacheDir, "pnpm-global")
	storeDir := filepath.Join(cacheDir, "pnpm-store")

	env := os.Environ()
	env = append(env, "PNPM_HOME="+binDir)
	env = append(env, "PNPM_CONFIG_GLOBAL_BIN_DIR="+binDir)
	env = append(env, "PNPM_CONFIG_GLOBAL_DIR="+globalDir)
	env = append(env, "PNPM_CONFIG_STORE_DIR="+storeDir)
	env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return env, nil
}

func run(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "pnpm", args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func build(ctx context.Context, dir string, env []string, cmdStr string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installFromSource(ctx context.Context, toolEnv []string, tool *config.PNPMTool) error {
	if tool.Package == "" {
		return fmt.Errorf("%w: pnpm tool %s", ErrMissingPackageURL, tool.Name)
	}
	if tool.SrcDir == "" {
		return fmt.Errorf("%w: pnpm tool %s", ErrMissingSrcDir, tool.Name)
	}
	repo, err := repoFromPackageURL(tool.Package)
	if err != nil {
		return err
	}
	sha := tool.SHA256
	if sha == "" {
		sha = tool.Checksum
	}
	dir, err := fetch.Repo(ctx, repo, tool.Version, sha)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", tool.Name, err)
	}

	// Run build steps.
	buildDir := filepath.Join(dir, tool.SrcDir)
	for _, cmd := range tool.Build {
		if err := build(ctx, buildDir, toolEnv, cmd); err != nil {
			return fmt.Errorf("%w: %w", ErrInstall, err)
		}
	}
	return nil
}

// repoFromPackageURL extracts the repository path (e.g.,
// "github.com/googleapis/google-cloud-node") from a GitHub archive URL
// like "https://github.com/googleapis/google-cloud-node/archive/<sha>.tar.gz".
func repoFromPackageURL(packageURL string) (string, error) {
	parts := strings.SplitN(packageURL, "/archive/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("%w: %q", ErrCannotExtractRepo, packageURL)
	}
	return strings.TrimPrefix(parts[0], "https://"), nil
}
