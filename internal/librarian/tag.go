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

package librarian

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
	"github.com/urfave/cli/v3"
)

// dartChangelogFile is created by `librarian bump` when preparing an initial
// Dart package release, distinguishing released libraries from newly generated
// unpublished libraries that already carry an initial version in librarian.yaml.
const dartChangelogFile = "CHANGELOG.md"

var (
	errMissingTagFormat           = errors.New("no tag format configured")
	errNoLibrariesAtReleaseCommit = errors.New("commit does not release any libraries")
)

func tagCommand() *cli.Command {
	return &cli.Command{
		Name:      "tag",
		Hidden:    true,
		Usage:     "tag a release commit based on the libraries published",
		UsageText: "librarian tag",
		Description: `tag creates git tags on a release commit, one tag per library that the
commit released, using the tag_format declared for each library in
librarian.yaml.

Run tag after librarian publish has succeeded. By default, the most
recent release commit reachable from HEAD is used; --release-commit
overrides this with a specific commit.

Examples:

	librarian tag
	librarian tag --release-commit=<sha>`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "release-commit",
				Usage: "the release commit to tag; default finds latest release commit",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return tag(ctx, cmd.String("release-commit"))
		},
	}
}

// tag implements the tag command. It finds the release commit to publish
// (unless already specified). The configuration at the release commit is used
// for all further operations.
func tag(ctx context.Context, releaseCommit string) error {
	if err := git.AssertGitStatusClean(ctx, command.Git); err != nil {
		return err
	}
	if releaseCommit == "" {
		var err error
		releaseCommit, err = findReleaseCommit(ctx)
		if err != nil {
			return err
		}
	}
	releaseCommitCfg, err := loadConfigAtRevision(ctx, releaseCommit)
	if err != nil {
		return err
	}
	if releaseCommitCfg.Default == nil || releaseCommitCfg.Default.TagFormat == "" {
		return errMissingTagFormat
	}
	librariesToTag, err := librariesToTagAtCommit(ctx, releaseCommit, releaseCommitCfg)
	if err != nil {
		return err
	}
	if len(librariesToTag) == 0 {
		return fmt.Errorf("error tagging %s: %w", releaseCommit, errNoLibrariesAtReleaseCommit)
	}

	tagFormat := releaseCommitCfg.Default.TagFormat
	for _, libraryToTag := range librariesToTag {
		lib, err := FindLibrary(releaseCommitCfg, libraryToTag)
		if err != nil {
			return err
		}
		tagName := git.FormatTagName(tagFormat, lib.Name, lib.Version)
		if err := git.Tag(ctx, command.Git, tagName, releaseCommit); err != nil {
			return fmt.Errorf("error creating tag %s: %w", tagName, err)
		}
	}
	return nil
}

func findReleaseCommit(ctx context.Context) (string, error) {
	headCfg, err := loadConfigAtRevision(ctx, "HEAD")
	if err == nil && headCfg.Language == config.LanguageDart {
		return findDartReleaseCommit(ctx, headCfg)
	}
	return findLatestReleaseCommitHash(ctx)
}

func librariesToTagAtCommit(ctx context.Context, releaseCommit string, releaseCommitCfg *config.Config) ([]string, error) {
	if releaseCommitCfg.Language == config.LanguageDart {
		localTags, err := git.Tags(ctx, command.Git)
		if err != nil {
			return nil, err
		}
		return findDartLibrariesToTag(ctx, releaseCommit, releaseCommitCfg, localTags)
	}
	beforeReleaseCommitCfg, err := loadConfigAtRevision(ctx, releaseCommit+"~")
	if err != nil {
		return nil, err
	}
	return findReleasedLibraries(beforeReleaseCommitCfg, releaseCommitCfg)
}

func findDartReleaseCommit(ctx context.Context, headCfg *config.Config) (string, error) {
	if headCfg.Default == nil || headCfg.Default.TagFormat == "" {
		return "", errMissingTagFormat
	}
	localTags, err := git.Tags(ctx, command.Git)
	if err != nil {
		return "", err
	}
	paths := append([]string{config.LibrarianYAML}, dartChangelogPaths(headCfg)...)
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
		released, err := findDartLibrariesToTag(ctx, commit, commitCfg, localTags)
		if err != nil {
			return "", err
		}
		if len(released) > 0 {
			return commit, nil
		}
	}
	return "", errReleaseCommitNotFound
}

func findDartLibrariesToTag(ctx context.Context, commit string, commitCfg *config.Config, localTags []string) ([]string, error) {
	beforeCfg, hasParentCfg, err := loadParentConfig(ctx, commit)
	if err != nil {
		return nil, err
	}
	releasedByCfg, err := findReleasedLibraries(beforeCfg, commitCfg)
	if err != nil {
		return nil, err
	}
	changelogsAtCommit, err := existingDartChangelogs(ctx, commit, commitCfg)
	if err != nil {
		return nil, err
	}
	var changelogsBefore map[string]bool
	if hasParentCfg {
		changelogsBefore, err = existingDartChangelogs(ctx, commit+"~", beforeCfg)
		if err != nil {
			return nil, err
		}
	}

	tagFormat := ""
	if commitCfg.Default != nil {
		tagFormat = commitCfg.Default.TagFormat
	}

	inCommitRelease := make(map[string]bool)
	for _, name := range releasedByCfg {
		lib, err := FindLibrary(commitCfg, name)
		if err != nil {
			return nil, err
		}
		if _, beforeErr := FindLibrary(beforeCfg, name); errors.Is(beforeErr, ErrLibraryNotFound) {
			if !changelogsAtCommit[name] {
				continue
			}
			if !hasParentCfg {
				tagName := git.FormatTagName(tagFormat, lib.Name, lib.Version)
				if slices.Contains(localTags, tagName) {
					continue
				}
			}
		}
		inCommitRelease[name] = true
	}

	untaggedWithChangelog := make(map[string]bool)
	for _, lib := range commitCfg.Libraries {
		if lib.SkipRelease || lib.Version == "" {
			continue
		}
		if err := semver.ValidateNext("", lib.Version); err != nil {
			return nil, err
		}
		if !changelogsAtCommit[lib.Name] {
			continue
		}
		tagName := git.FormatTagName(tagFormat, lib.Name, lib.Version)
		if !slices.Contains(localTags, tagName) {
			untaggedWithChangelog[lib.Name] = true
		}
		if inCommitRelease[lib.Name] || !hasParentCfg {
			continue
		}
		beforeLib, beforeErr := FindLibrary(beforeCfg, lib.Name)
		unskippedInCommit := beforeErr == nil && beforeLib.SkipRelease
		if !changelogsBefore[lib.Name] || unskippedInCommit {
			inCommitRelease[lib.Name] = true
		}
	}

	if len(inCommitRelease) == 0 {
		return nil, nil
	}

	var results []string
	for _, lib := range commitCfg.Libraries {
		if inCommitRelease[lib.Name] || untaggedWithChangelog[lib.Name] {
			results = append(results, lib.Name)
		}
	}
	return results, nil
}

func existingDartChangelogs(ctx context.Context, revision string, cfg *config.Config) (map[string]bool, error) {
	paths := dartChangelogPaths(cfg)
	if len(paths) == 0 {
		return nil, nil
	}
	existingFiles, err := git.ListFilesAtRevision(ctx, command.Git, revision, paths...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(existingFiles))
	for _, lib := range cfg.Libraries {
		if slices.Contains(existingFiles, dartChangelogPath(cfg, lib)) {
			result[lib.Name] = true
		}
	}
	return result, nil
}

func dartChangelogPaths(cfg *config.Config) []string {
	var paths []string
	for _, lib := range cfg.Libraries {
		p := dartChangelogPath(cfg, lib)
		if p != "" && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	return paths
}

func dartChangelogPath(cfg *config.Config, lib *config.Library) string {
	output := filepath.ToSlash(libraryOutput(config.LanguageDart, lib, cfg.Default))
	return path.Join(output, dartChangelogFile)
}

func loadConfigAtRevision(ctx context.Context, revision string) (*config.Config, error) {
	content, err := git.ShowFileAtRevision(ctx, command.Git, revision, config.LibrarianYAML)
	if err != nil {
		return nil, err
	}
	return yaml.Unmarshal[config.Config]([]byte(content))
}

func loadParentConfig(ctx context.Context, commit string) (*config.Config, bool, error) {
	cfg, err := loadConfigAtRevision(ctx, commit+"~")
	if err != nil {
		if errors.Is(err, git.ErrGitShow) {
			return &config.Config{}, false, nil
		}
		return nil, false, err
	}
	return cfg, true, nil
}
