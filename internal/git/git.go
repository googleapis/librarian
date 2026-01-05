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

// Package git provides functions for determining changes in a git repository.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/googleapis/librarian/internal/command"
)

var (
	// errGitShow is included in any error returned by [GitShowFile].
	errGitShow = errors.New("failed to show file")
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
func GetLastTag(ctx context.Context, gitExe, remote, branch string) (string, error) {
	ref := fmt.Sprintf("%s/%s", remote, branch)
	cmd := exec.CommandContext(ctx, gitExe, "describe", "--abbrev=0", "--tags", ref)
	contents, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get last tag for repo %s: %w\noutput: %s", ref, err, string(contents))
	}
	tag := string(contents)
	return strings.TrimSuffix(tag, "\n"), nil
}

// FilesChangedSince returns the files changed since the given git ref.
func FilesChangedSince(ctx context.Context, ref, gitExe string, ignoredChanges []string) ([]string, error) {
	cmd := exec.CommandContext(ctx, gitExe, "diff", "--name-only", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get files changed since tag %s: %w\noutput: %s", ref, err, string(output))
	}
	return filesFilter(ignoredChanges, strings.Split(string(output), "\n")), nil
}

func filesFilter(ignoredChanges []string, files []string) []string {
	var patterns []gitignore.Pattern
	for _, p := range ignoredChanges {
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.HasPrefix(output, []byte(" create mode "))
}

// GitVersion checks the git version.
func GitVersion(ctx context.Context, gitExe string) error {
	return command.Run(ctx, gitExe, "--version")
}

// GitRemoteURL checks the git remote URL.
func GitRemoteURL(ctx context.Context, gitExe, remote string) error {
	return command.Run(ctx, gitExe, "remote", "get-url", remote)
}

// GitShowFile shows the contents of the file found at the given path on the
// given remote/branch.
func GitShowFile(ctx context.Context, gitExe, remote, branch, path string) (string, error) {
	remoteBranchPath := fmt.Sprintf("%s/%s:%s", remote, branch, path)
	cmd := exec.CommandContext(ctx, gitExe, "show", remoteBranchPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w %s: %w\noutput: %s", errGitShow, remoteBranchPath, err, string(output))
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

// MatchesBranchPoint returns an error if the local repository has unpushed changes.
func MatchesBranchPoint(ctx context.Context, gitExe, remote, branch string) error {
	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	delta := fmt.Sprintf("%s...HEAD", remoteBranch)
	cmd := exec.CommandContext(ctx, gitExe, "diff", "--name-only", delta)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to diff against branch %s: %w\noutput: %s", remoteBranch, err, string(output))
	}
	if len(output) != 0 {
		return fmt.Errorf("the local repository does not match its branch point from %s, change files:\n%s", remoteBranch, string(output))
	}
	return nil
}

// ChangesInDirectorySinceTag checks if there have been any changes in directory since tag.
func ChangesInDirectorySinceTag(ctx context.Context, gitExe, tag, dir string) (int, error) {
	delta := fmt.Sprintf("%s...HEAD", tag)
	cmd := exec.CommandContext(ctx, gitExe, "rev-list", "--count", delta, "--", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("running %s failed: %w\noutput: %s", cmd.String(), err, string(output))
	}
	return strconv.Atoi(strings.TrimSpace(string(output)))
}
