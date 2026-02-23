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
	"strings"

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
		Usage:     "tags a release commit based on the libraries published",
		UsageText: "librarian tag",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "release-commit",
				Usage: "the release commit to tag; default finds latest release commit",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			return tag(ctx, cfg, cmd.String("release-commit"))
		},
	}
}

// tag implements the tag command. It is provided with the configuration
// at HEAD, just to find the git executable to use, after which it finds the
// release commit to publish (unless already specified). The configuration at
// the release commit is used for all further operations.
func tag(ctx context.Context, cfg *config.Config, releaseCommit string) error {
	gitExe := "git"
	if cfg.Release != nil {
		gitExe = command.GetExecutablePath(cfg.Release.Preinstalled, "git")
	}
	if err := git.AssertGitStatusClean(ctx, gitExe); err != nil {
		return err
	}
	if releaseCommit == "" {
		var err error
		releaseCommit, err = findLatestReleaseCommitHash(ctx, gitExe)
		if err != nil {
			return err
		}
	}
	releaseCommitCfgContent, err := git.ShowFileAtRevision(ctx, gitExe, releaseCommit, librarianConfigPath)
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
	beforeReleaseCommitCfgContent, err := git.ShowFileAtRevision(ctx, gitExe, releaseCommit+"~", librarianConfigPath)
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
		tagName := formatTagName(tagFormat, lib)
		err = git.Tag(ctx, gitExe, tagName, releaseCommit)
		if err != nil {
			return fmt.Errorf("error creating tag %s: %w", tagName, err)
		}
	}
	return nil
}

// formatTagName computes the name of the tag expected to be applied to the
// commit that released the given library.
func formatTagName(tagFormat string, lib *config.Library) string {
	return strings.NewReplacer("{name}", lib.Name, "{version}", lib.Version).Replace(tagFormat)
}
