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

package swift

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
)

const (
	minSwiftMajor = 6
	minSwiftMinor = 2
	toolsDir      = "swift_tools"
)

var (
	errCannotParseSwiftVersion = errors.New("failed to parse swift version")
	errInvalidTool             = errors.New("invalid tool configuration")
	errMissingExecutable       = errors.New("is not installed or not in PATH, which is required for Swift tool installation")
	errSwiftVersionTooLow      = fmt.Errorf("swift version is less than the required minimum version %d.%d", minSwiftMajor, minSwiftMinor)
	swiftVersionRegex          = regexp.MustCompile(`(?i)swift\s+version\s+(\d+)\.(\d+)`)
)

// Install installs the tools required for Swift library generation.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Swift) == 0 {
		return nil
	}
	if err := verifyPrerequisites(ctx, tools.Swift); err != nil {
		return err
	}
	bin, err := binDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	for _, tool := range tools.Swift {
		if err := installTool(ctx, tool, bin); err != nil {
			return err
		}
	}
	return nil
}

// InstallDir returns the directory where Swift tools are installed.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, toolsDir))
}

// binDir returns the bin directory where Swift executables are stored.
func binDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, "bin"), nil
}

// toolsEnv returns an environment map with `PATH` set to the Swift tools bin directory.
//
// This is combined with the system PATH in `command.RunWithEnv()`, a strange idiom in my (coryan)
// opinion, but seemingly consistent with the rest of the code in librarian.
func toolsEnv() (map[string]string, error) {
	bin, err := binDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{"PATH": bin}, nil
}

func verifyPrerequisites(ctx context.Context, tools []*config.SwiftTool) error {
	for _, cmd := range []string{command.Swift, "swift-format"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s %w: %w", cmd, errMissingExecutable, err)
		}
	}
	for _, tool := range tools {
		if tool.Name == "" {
			return fmt.Errorf("%w: name must be specified: %+v", errInvalidTool, tool)
		}
		hasLocal := tool.LocalPath != ""
		hasRemote := tool.Repo != "" || tool.Version != ""
		if hasLocal && hasRemote {
			return fmt.Errorf("%w: cannot specify both local_path and repo/version: %+v", errInvalidTool, tool)
		}
		if !hasLocal && !hasRemote {
			return fmt.Errorf("%w: must specify either local_path or repo and version: %+v", errInvalidTool, tool)
		}
		if hasRemote {
			if tool.Repo == "" {
				return fmt.Errorf("%w: repo must be specified: %+v", errInvalidTool, tool)
			}
			if tool.Version == "" {
				return fmt.Errorf("%w: version must be specified: %+v", errInvalidTool, tool)
			}
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git %w: %w", errMissingExecutable, err)
			}
		}
	}
	return verifySwiftVersion(ctx)
}

func verifySwiftVersion(ctx context.Context) error {
	output, err := command.Output(ctx, command.Swift, "--version")
	if err != nil {
		return fmt.Errorf("failed to get swift version: %w", err)
	}
	return checkSwiftVersionOutput(output)
}

func checkSwiftVersionOutput(output string) error {
	matches := swiftVersionRegex.FindStringSubmatch(output)
	if len(matches) < 3 {
		return fmt.Errorf("%w: %q", errCannotParseSwiftVersion, output)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%w: invalid major version in %q", errCannotParseSwiftVersion, output)
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("%w: invalid minor version in %q", errCannotParseSwiftVersion, output)
	}
	if major < minSwiftMajor || (major == minSwiftMajor && minor < minSwiftMinor) {
		return fmt.Errorf("%w: found %d.%d, required >= %d.%d", errSwiftVersionTooLow, major, minor, minSwiftMajor, minSwiftMinor)
	}
	return nil
}

func installTool(ctx context.Context, tool *config.SwiftTool, bin string) error {
	var buildDir string
	if tool.LocalPath != "" {
		absPath, err := filepath.Abs(tool.LocalPath)
		if err != nil {
			return fmt.Errorf("failed to resolve local path for %s: %w", tool.Name, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("local path %q does not exist: %w", absPath, err)
		}
		buildDir = absPath
	} else {
		tmpDir, err := os.MkdirTemp("", "swift-tool-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory for %s: %w", tool.Name, err)
		}
		defer os.RemoveAll(tmpDir)
		repoURL := formatRepoURL(tool.Repo)
		if err := command.Run(ctx, "git", "clone", "--depth", "1", "--branch", tool.Version, repoURL, tmpDir); err != nil {
			return fmt.Errorf("failed to clone %s from %s: %w", tool.Name, repoURL, err)
		}
		buildDir = tmpDir
	}

	buildArgs := []string{"build", "-c", "release"}
	if tool.Product != "" {
		buildArgs = append(buildArgs, "--product", tool.Product)
	}
	if err := command.RunInDir(ctx, buildDir, command.Swift, buildArgs...); err != nil {
		return fmt.Errorf("failed to build %s: %w", tool.Name, err)
	}

	output, err := command.OutputInDir(ctx, buildDir, command.Swift, "build", "-c", "release", "--show-bin-path")
	if err != nil {
		return fmt.Errorf("failed to get bin path: %w", err)
	}
	srcBinary := filepath.Join(strings.TrimSpace(output), tool.Name)
	destBinary := filepath.Join(bin, tool.Name)
	return copyExecutable(srcBinary, destBinary)
}

func formatRepoURL(repo string) string {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	if strings.HasPrefix(repo, "github.com/") {
		return "https://" + repo
	}
	return "https://github.com/" + repo
}

func copyExecutable(src, dst string) error {
	if err := filesystem.CopyFile(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}
