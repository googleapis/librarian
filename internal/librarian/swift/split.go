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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/git"
)

var (
	// DefaultRootFiles are the root files preserved on every commit in the split repository.
	DefaultRootFiles = []string{"LICENSE", "CODE_OF_CONDUCT.md", "CONTRIBUTING.md"}
)

// SplitParams holds parameters for splitting a subdirectory out of the monorepo.
type SplitParams struct {
	// TargetDir is the directory path relative to the monorepo root (e.g. packages/auth, generated/google-rpc).
	TargetDir string
	// Origin is the source commit or branch to split from (default: HEAD).
	Origin string
	// RootFiles is the list of root files to preserve across every commit.
	RootFiles []string
	// GitExe is the path to the git binary (default: command.Git).
	GitExe string
}

// Split splits a subdirectory from the repository into a standalone commit history,
// placing the subtree contents at the root and preserving specified root files on every commit.
// It returns the resulting 40-character commit hash.
func Split(ctx context.Context, params SplitParams) (string, error) {
	gitExe := params.GitExe
	if gitExe == "" {
		gitExe = command.Git
	}
	origin := params.Origin
	if origin == "" {
		origin = "HEAD"
	}
	targetDir := strings.Trim(params.TargetDir, "/")
	if targetDir == "" {
		return "", errors.New("target directory cannot be empty")
	}

	rawSHA, err := git.SubtreeSplit(ctx, gitExe, targetDir, origin)
	if err != nil {
		return "", err
	}

	rootFiles := params.RootFiles
	if rootFiles == nil {
		rootFiles = DefaultRootFiles
	}
	if len(rootFiles) == 0 {
		return rawSHA, nil
	}

	return rewriteHistoryWithRootFiles(ctx, gitExe, rawSHA, origin, rootFiles)
}

func rewriteHistoryWithRootFiles(ctx context.Context, gitExe, rawSHA, origin string, rootFiles []string) (string, error) {
	rootEntries, err := getRootEntries(ctx, gitExe, origin, rootFiles)
	if err != nil {
		return "", err
	}
	if len(rootEntries) == 0 {
		return rawSHA, nil
	}

	revOutput, err := command.Output(ctx, gitExe, "rev-list", "--reverse", "--topo-order", rawSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get commit list for %s: %w", rawSHA, err)
	}
	commits := strings.Fields(revOutput)
	if len(commits) == 0 {
		return rawSHA, nil
	}

	commitMap := make(map[string]string, len(commits))
	lastNewCommit := rawSHA

	for _, c := range commits {
		treeOut, err := command.Output(ctx, gitExe, "ls-tree", c)
		if err != nil {
			return "", fmt.Errorf("failed to ls-tree for %s: %w", c, err)
		}

		var currentEntries []string
		for line := range strings.SplitSeq(treeOut, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				filename := parts[len(parts)-1]
				if slices.Contains(rootFiles, filename) {
					continue
				}
			}
			currentEntries = append(currentEntries, line)
		}

		// Git tree objects require entries to be sorted alphabetically, with tree
		// objects sorted as if they have a trailing slash. Unsorted entries cause git fsck to fail.
		combined := append(currentEntries, rootEntries...)
		slices.SortFunc(combined, func(a, b string) int {
			partsA := strings.SplitN(a, "\t", 2)
			partsB := strings.SplitN(b, "\t", 2)
			if len(partsA) < 2 || len(partsB) < 2 {
				return strings.Compare(a, b)
			}
			nameA, nameB := partsA[1], partsB[1]
			isTreeA := strings.Contains(partsA[0], " tree ")
			isTreeB := strings.Contains(partsB[0], " tree ")
			if isTreeA {
				nameA += "/"
			}
			if isTreeB {
				nameB += "/"
			}
			return strings.Compare(nameA, nameB)
		})
		mktreeInput := strings.Join(combined, "\n") + "\n"

		newTree, err := command.OutputWithStdin(ctx, strings.NewReader(mktreeInput), gitExe, "mktree")
		if err != nil {
			return "", fmt.Errorf("git mktree failed for %s: %w", c, err)
		}
		newTree = strings.TrimSpace(newTree)

		parentsOut, err := command.Output(ctx, gitExe, "log", "-n", "1", "--pretty=%P", c)
		if err != nil {
			return "", fmt.Errorf("failed to get parents for %s: %w", c, err)
		}
		var parentArgs []string
		for p := range strings.FieldsSeq(parentsOut) {
			mappedP := p
			if mapped, ok := commitMap[p]; ok {
				mappedP = mapped
			}
			parentArgs = append(parentArgs, "-p", mappedP)
		}

		metaOut, err := command.Output(ctx, gitExe, "log", "-n", "1", "--pretty=format:%an%x00%ae%x00%ad%x00%cn%x00%ce%x00%cd%x00%B", c)
		if err != nil {
			return "", err
		}
		parts := strings.SplitN(metaOut, "\x00", 7)
		if len(parts) < 7 {
			return "", fmt.Errorf("failed to parse metadata for commit %s", c)
		}
		authorName, authorEmail, authorDate := parts[0], parts[1], parts[2]
		committerName, committerEmail, committerDate := parts[3], parts[4], parts[5]
		commitMsg := parts[6]

		env := map[string]string{
			"GIT_AUTHOR_NAME":     authorName,
			"GIT_AUTHOR_EMAIL":    authorEmail,
			"GIT_AUTHOR_DATE":     authorDate,
			"GIT_COMMITTER_NAME":  committerName,
			"GIT_COMMITTER_EMAIL": committerEmail,
			"GIT_COMMITTER_DATE":  committerDate,
		}
		args := []string{"-c", "commit.gpgsign=false", "commit-tree", newTree}
		args = append(args, parentArgs...)

		newCommit, err := command.OutputWithStdinAndEnv(ctx, strings.NewReader(commitMsg), env, gitExe, args...)
		if err != nil {
			return "", fmt.Errorf("git commit-tree failed for %s: %w", c, err)
		}
		newCommit = strings.TrimSpace(newCommit)
		commitMap[c] = newCommit
		lastNewCommit = newCommit
	}

	return lastNewCommit, nil
}

func getRootEntries(ctx context.Context, gitExe, origin string, rootFiles []string) ([]string, error) {
	var entries []string
	for _, f := range rootFiles {
		out, err := command.Output(ctx, gitExe, "ls-tree", origin, "--", f)
		if err != nil {
			return nil, fmt.Errorf("failed to ls-tree for %s at %s: %w", f, origin, err)
		}
		for line := range strings.SplitSeq(out, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				entries = append(entries, line)
			}
		}
	}
	return entries, nil
}
