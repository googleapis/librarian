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

// Package change provides functions for determining changes in a git repository.
package change

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// AssertGitStatusClean returns an error if the git working directory has uncommitted changes.
func AssertGitStatusClean(ctx context.Context, git string) error {
	cmd := exec.CommandContext(ctx, git, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if len(output) > 0 {
		return fmt.Errorf("git working directory is not clean")
	}
	return nil
}

// GetLastTag returns the last git tag for the given release configuration.
func GetLastTag(ctx context.Context, cfg *config.Release) (string, error) {
	branch := fmt.Sprintf("%s/%s", cfg.Remote, cfg.Branch)
	cmd := exec.CommandContext(ctx, cfg.GetExecutablePath("git"), "describe", "--abbrev=0", "--tags", branch)
	cmd.Dir = "."
	contents, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	tag := string(contents)
	return strings.TrimSuffix(tag, "\n"), nil
}

// FilesChangedSince returns the files changed since the given git ref.
func FilesChangedSince(ctx context.Context, ref string, cfg *config.Release) ([]string, error) {
	cmd := exec.CommandContext(ctx, cfg.GetExecutablePath("git"), "diff", "--name-only", ref)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return filesFilter(cfg, strings.Split(string(output), "\n")), nil
}

func filesFilter(cfg *config.Release, files []string) []string {
	var patterns []gitignore.Pattern
	if cfg == nil {
		return files
	}
	for _, p := range cfg.IgnoredChanges {
		patterns = append(patterns, gitignore.ParsePattern(p, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	files = slices.DeleteFunc(files, func(a string) bool {
		if a == "" {
			return true
		}
		return matcher.Match(strings.Split(a, "/"), false)
	})
	return files
}

// IsNewFile returns true if the given file is new since the given git ref.
func IsNewFile(ctx context.Context, gitExe, ref, name string) bool {
	delta := fmt.Sprintf("%s..HEAD", ref)
	cmd := exec.CommandContext(ctx, gitExe, "diff", "--summary", delta, "--", name)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	expected := fmt.Sprintf(" create mode 100644 %s", name)
	return bytes.HasPrefix(output, []byte(expected))
}

// GitVersion checks the git version.
func GitVersion(ctx context.Context, gitExe string) error {
	return command.Run(ctx, gitExe, "--version")
}

// GitRemoteURL checks the git remote URL.
func GitRemoteURL(ctx context.Context, gitExe, remote string) error {
	return command.Run(ctx, gitExe, "remote", "get-url", remote)
}
