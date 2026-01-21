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

package librarian

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Usage:     "publishes client libraries",
		UsageText: "librarian publish",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "execute",
				Usage: "fully publish (default is to only perform a dry run)",
			},
			&cli.StringFlag{
				Name:  "library",
				Usage: "library to find a release commit for; default finds latest release commit for any library",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print commands without executing (legacy Rust-only flag)",
			},
			&cli.BoolFlag{
				Name:  "dry-run-keep-going",
				Usage: "print commands without executing, don't stop on error (legacy Rust-only flag)",
			},
			&cli.BoolFlag{
				Name:  "skip-semver-checks",
				Usage: "skip semantic versioning checks (legacy Rust-only flag)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			if cfg.Language == languageRust {
				return legacyRustPublish(ctx, cfg, cmd)
			}
			return publish(ctx, cfg, cmd.String("library"), cmd.Bool("execute"))
		},
	}
}

// legacyRustPublish implements the legacy publish behavior while new publish
// behavior is being implemented.
// TODO(https://github.com/googleapis/librarian/issues/3600): align flags
// with new design and remove this function once Rust has migrated.
func legacyRustPublish(ctx context.Context, cfg *config.Config, cmd *cli.Command) error {
	dryRun := cmd.Bool("dry-run")
	skipSemverChecks := cmd.Bool("skip-semver-checks")
	dryRunKeepGoing := cmd.Bool("dry-run-keep-going")
	return rust.Publish(ctx, cfg.Release, dryRun, dryRunKeepGoing, skipSemverChecks)
}

// publish implements the publish command. It is provided with the configuration
// at HEAD, just to find the git executable to use, after which it finds the
// release commit to publish. The configuration at the release commit is used
// for all further operations (and the repo will be checked out at that commit).
// The library flag allows a user to identify a specific release to publish, in
// case of overlapping releases being performed. The execute flag says whether to
// actually publish (true) or just perform a dry run (false).
func publish(ctx context.Context, cfg *config.Config, library string, execute bool) error {
	gitExe := "git"
	if cfg.Release != nil {
		gitExe = command.GetExecutablePath(cfg.Release.Preinstalled, "git")
	}
	if err := git.AssertGitStatusClean(ctx, gitExe); err != nil {
		return err
	}
	releaseCommitHash, err := findLatestReleaseCommitHash(ctx, gitExe, library)
	if err != nil {
		return err
	}
	if err := git.Checkout(ctx, gitExe, releaseCommitHash); err != nil {
		return err
	}
	// Reload the config after checking out the release commit.
	cfg, err = yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	// Load the immediately-preceding config so we can find all libraries that
	// were released by that commit. (This duplicates work done in
	// findLatestReleaseCommitHash, but keeps the interface simple - and means
	// that if we want to be able to specify the release commit directly, we
	// can skip findLatestReleaseCommitHash entirely.)
	cfgContentBeforeCommit, err := git.ShowFileAtRevision(ctx, gitExe, "HEAD~", librarianConfigPath)
	if err != nil {
		return err
	}
	cfgBeforeReleaseCommit, err := yaml.Unmarshal[config.Config]([]byte(cfgContentBeforeCommit))
	if err != nil {
		return err
	}
	librariesToPublish, err := findReleasedLibraries(cfgBeforeReleaseCommit, cfg)
	if err != nil {
		return err
	}

	switch cfg.Language {
	case languageFake:
		return fakePublish(librariesToPublish, execute)
	default:
		return fmt.Errorf("publish not implemented for %q", cfg.Language)
	}
}
