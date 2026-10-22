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

package dart

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/yaml"
)

// changelogFile is created by `librarian bump` when preparing an initial Dart
// package release, distinguishing released libraries from newly generated
// unpublished libraries that already carry version 0.1.0 in librarian.yaml.
const changelogFile = "CHANGELOG.md"

// FindReleaseCommit returns the latest commit hash reachable from HEAD that
// released one or more Dart libraries, or ("", nil) if none is found.
func FindReleaseCommit(ctx context.Context, headCfg *config.Config) (string, error) {
	paths := append([]string{config.LibrarianYAML}, changelogPaths(headCfg)...)
	commits, err := git.FindCommitsForPaths(ctx, command.Git, paths...)
	if err != nil {
		return "", err
	}
	for _, commit := range commits {
		commitCfg, err := loadConfigAtRevision(ctx, commit)
		if err != nil {
			if errors.Is(err, git.ErrGitShow) {
				continue
			}
			return "", err
		}
		released, err := LibrariesToTag(ctx, commit, commitCfg)
		if err != nil {
			if errors.Is(err, git.ErrGitShow) {
				continue
			}
			return "", err
		}
		if len(released) > 0 {
			return commit, nil
		}
	}
	return "", nil
}

// LibrariesToTag returns the names of Dart libraries released by commit
// compared against commit~.
func LibrariesToTag(ctx context.Context, commit string, commitCfg *config.Config) ([]string, error) {
	beforeCfg, err := loadConfigAtRevision(ctx, commit+"~")
	if err != nil {
		return nil, err
	}
	changelogsAtCommit, err := existingChangelogs(ctx, commit, commitCfg)
	if err != nil {
		return nil, err
	}
	changelogsBefore, err := existingChangelogs(ctx, commit+"~", beforeCfg)
	if err != nil {
		return nil, err
	}

	beforeByName := make(map[string]*config.Library, len(beforeCfg.Libraries))
	for _, lib := range beforeCfg.Libraries {
		beforeByName[lib.Name] = lib
	}

	var results []string
	for _, lib := range commitCfg.Libraries {
		if lib.SkipRelease {
			continue
		}
		beforeLib, existedBefore := beforeByName[lib.Name]
		if lib.Version == "" {
			if existedBefore && beforeLib.Version != "" {
				return nil, fmt.Errorf("library %q has no version; was at version %q", lib.Name, beforeLib.Version)
			}
			continue
		}
		if !existedBefore || beforeLib.Version == "" {
			if err := semver.ValidateNext("", lib.Version); err != nil {
				return nil, err
			}
			if !changelogsAtCommit[lib.Name] {
				continue
			}
			results = append(results, lib.Name)
			continue
		}
		if lib.Version != beforeLib.Version {
			if err := semver.ValidateNext(beforeLib.Version, lib.Version); err != nil {
				return nil, err
			}
			results = append(results, lib.Name)
			continue
		}
		if changelogsAtCommit[lib.Name] && (!changelogsBefore[lib.Name] || beforeLib.SkipRelease) {
			results = append(results, lib.Name)
		}
	}
	return results, nil
}

func existingChangelogs(ctx context.Context, revision string, cfg *config.Config) (map[string]bool, error) {
	paths := changelogPaths(cfg)
	if len(paths) == 0 {
		return nil, nil
	}
	existingFiles, err := git.ListFilesAtRevision(ctx, command.Git, revision, paths...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(existingFiles))
	for _, lib := range cfg.Libraries {
		if slices.Contains(existingFiles, changelogPath(cfg, lib)) {
			result[lib.Name] = true
		}
	}
	return result, nil
}

func changelogPaths(cfg *config.Config) []string {
	var paths []string
	for _, lib := range cfg.Libraries {
		p := changelogPath(cfg, lib)
		if p != "" && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	return paths
}

func changelogPath(cfg *config.Config, lib *config.Library) string {
	output := filepath.ToSlash(libraryOutput(lib, cfg.Default))
	return path.Join(output, changelogFile)
}

func loadConfigAtRevision(ctx context.Context, revision string) (*config.Config, error) {
	content, err := git.ShowFileAtRevision(ctx, command.Git, revision, config.LibrarianYAML)
	if err != nil {
		return nil, err
	}
	return yaml.Unmarshal[config.Config]([]byte(content))
}
