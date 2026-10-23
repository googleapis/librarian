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

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

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
		commit, err := dart.FindReleaseCommit(ctx, headCfg)
		if err != nil {
			return "", err
		}
		if commit == "" {
			return "", errReleaseCommitNotFound
		}
		return commit, nil
	}
	return findLatestReleaseCommitHash(ctx)
}

func librariesToTagAtCommit(ctx context.Context, releaseCommit string, releaseCommitCfg *config.Config) ([]string, error) {
	if releaseCommitCfg.Language == config.LanguageDart {
		return dart.LibrariesToTag(ctx, releaseCommit, releaseCommitCfg)
	}
	beforeReleaseCommitCfg, err := loadConfigAtRevision(ctx, releaseCommit+"~")
	if err != nil {
		return nil, err
	}
	return findReleasedLibraries(beforeReleaseCommitCfg, releaseCommitCfg)
}

func loadConfigAtRevision(ctx context.Context, revision string) (*config.Config, error) {
	content, err := git.ShowFileAtRevision(ctx, command.Git, revision, config.LibrarianYAML)
	if err != nil {
		return nil, err
	}
	return yaml.Unmarshal[config.Config]([]byte(content))
}
