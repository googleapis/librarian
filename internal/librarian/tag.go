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
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errNoLibrariesAtReleaseCommit = errors.New("commit does not release any libraries")

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
		latestReleaseCommit, err := findLatestReleaseCommitHash(ctx)
		if err != nil {
			return err
		}
		releaseCommit = latestReleaseCommit
	}
	releaseCommitCfgContent, err := git.ShowFileAtRevision(ctx, command.Git, releaseCommit, config.LibrarianYAML)
	if err != nil {
		return err
	}
	releaseCommitCfg, err := yaml.Unmarshal[config.Config]([]byte(releaseCommitCfgContent))
	if err != nil {
		return err
	}
	// Load the immediately-preceding config so we can find all libraries that
	// were released by that commit. (This duplicates work done in
	// findLatestReleaseCommitHash, but keeps the interface simple - and means
	// that if we specify the release commit directly, we can skip
	// findLatestReleaseCommitHash entirely.)
	beforeReleaseCommitCfgContent, err := git.ShowFileAtRevision(ctx, command.Git, releaseCommit+"~", config.LibrarianYAML)
	if err != nil {
		return err
	}
	beforeReleaseCommitCfg, err := yaml.Unmarshal[config.Config]([]byte(beforeReleaseCommitCfgContent))
	if err != nil {
		return err
	}
	librariesToTag, err := findReleasedLibraries(beforeReleaseCommitCfg, releaseCommitCfg)
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
		err = git.Tag(ctx, command.Git, tagName, releaseCommit)
		if err != nil {
			return fmt.Errorf("error creating tag %s: %w", tagName, err)
		}
	}
	return nil
}
