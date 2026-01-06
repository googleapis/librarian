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
				Name:  "dry-run",
				Usage: "print commands without executing",
			},
			&cli.BoolFlag{
				Name:  "skip-semver-checks",
				Usage: "skip semantic versioning checks",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			dryRun := cmd.Bool("dry-run")
			skipSemverChecks := cmd.Bool("skip-semver-checks")
			return publish(ctx, cfg, dryRun, skipSemverChecks)
		},
	}
}

func publish(ctx context.Context, cfg *config.Config, dryRun bool, skipSemverChecks bool) error {
	if err := verifyRequiredTools(ctx, cfg.Language, cfg.Release); err != nil {
		return err
	}
	gitExe := "git"
	if cfg.Release != nil {
		gitExe = command.GetExecutablePath(cfg.Release.Preinstalled, "git")
	}
	if err := git.AssertGitStatusClean(ctx, gitExe); err != nil {
		return err
	}
	lastTag, err := git.GetLastTag(ctx, gitExe, cfg.Release.Remote, cfg.Release.Branch)
	if err != nil {
		return err
	}
	files, err := git.FilesChangedSince(ctx, lastTag, gitExe, cfg.Release.IgnoredChanges)
	if err != nil {
		return err
	}
	switch cfg.Language {
	case languageFake:
		return fakePublish()
	case languageRust:
		return rust.PublishCrates(ctx, cfg.Release, dryRun, skipSemverChecks, lastTag, files)
	default:
		return fmt.Errorf("publish not implemented for %q", cfg.Language)
	}
}

// verifyRequiredTools verifies all the necessary tools are installed.
func verifyRequiredTools(ctx context.Context, language string, cfg *config.Release) error {
	switch language {
	case languageFake:
		return nil
	case languageRust:
		return rust.PreFlight(ctx, cfg.Preinstalled, cfg.Remote, cfg.Tools["cargo"])
	default:
		return fmt.Errorf("unknown language: %s", language)
	}
}
