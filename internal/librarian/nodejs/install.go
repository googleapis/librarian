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

package nodejs

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

const (
	// gapicGeneratorSubdir is the sub-directory within the
	// google-cloud-node repo that contains the gapic-generator-typescript
	// source.
	gapicGeneratorSubdir = "core/generator/gapic-generator-typescript"

	toolsDir = "nodejs_tools"
)

var (
	errNoToolsSpecified  = errors.New("no tools.pnpm field specified in configuration")
	errCannotExtractRepo = errors.New("cannot extract repo from package URL")
	errMissingExecutable = errors.New("is not installed or not in PATH, which is required for Node.js tool installation")
	errMissingPackageURL = errors.New("has build steps but no package URL")
)

// Install installs Node.js tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.PNPM) == 0 {
		return errNoToolsSpecified
	}

	for _, cmd := range []string{"node", "pnpm"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s %w: %w", cmd, errMissingExecutable, err)
		}
	}

	env, err := getPNPMEnv()
	if err != nil {
		return err
	}

	for _, tool := range tools.PNPM {
		if len(tool.Build) > 0 {
			if err := installPNPMToolFromSource(ctx, env, tool); err != nil {
				return err
			}
			continue
		}

		pkg := tool.Package
		if pkg == "" {
			pkg = fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		}
		if err := runPNPM(ctx, "", env, "add", "-g", pkg); err != nil {
			return err
		}
	}
	return nil
}

// InstallDir gets the directory where tools should be installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, toolsDir))
}

// getToolsEnv returns an environment map with the Node.js tools bin directory prepended to PATH.
func getToolsEnv() (map[string]string, error) {
	binDir, err := InstallDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{"PATH": binDir}, nil
}

// getPNPMEnv constructs a transient environment variable block to configure pnpm.
//
// This redirects all globally-installed pnpm binaries to LIBRARIAN_BIN, and
// virtual stores / content-addressable storage caches to LIBRARIAN_CACHE.
// This enables complete environment caching and restore on CI runners,
// while permanently avoiding persistent side-effects on the host machine
// (it does not modify the user's personal ~/.config/pnpm/rc files).
func getPNPMEnv() ([]string, error) {
	cacheDir, err := cache.Directory()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve librarian cache directory: %w", err)
	}
	binDir, err := InstallDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve librarian bin directory: %w", err)
	}

	env := os.Environ()
	env = append(env, "PNPM_HOME="+binDir)
	env = append(env, "PNPM_CONFIG_GLOBAL_BIN_DIR="+binDir)
	env = append(env, "PNPM_CONFIG_GLOBAL_DIR="+filepath.Join(cacheDir, "pnpm-global"))
	env = append(env, "PNPM_CONFIG_STORE_DIR="+filepath.Join(cacheDir, "pnpm-store"))
	env = append(env, "PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS=true")
	env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return env, nil
}

func runPNPM(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "pnpm", args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runPNPMBuildCmd(ctx context.Context, dir string, env []string, cmdStr string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installPNPMToolFromSource(ctx context.Context, env []string, tool *config.PNPMTool) error {
	if tool.Package == "" {
		return fmt.Errorf("pnpm tool %s %w", tool.Name, errMissingPackageURL)
	}
	repo, err := repoFromPackageURL(tool.Package)
	if err != nil {
		return err
	}
	dir, err := fetch.Repo(ctx, repo, tool.Version, tool.Checksum)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", tool.Name, err)
	}

	// Run build steps.
	genDir := filepath.Join(dir, gapicGeneratorSubdir)
	for _, cmd := range tool.Build {
		if err := runPNPMBuildCmd(ctx, genDir, env, cmd); err != nil {
			return err
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
		return "", fmt.Errorf("%w %q", errCannotExtractRepo, packageURL)
	}
	return strings.TrimPrefix(parts[0], "https://"), nil
}
