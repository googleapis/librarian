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
	librarianrust "github.com/googleapis/librarian/internal/librarian/rust"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust_release"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	rustPublishCrates  = sidekickrust.PublishCrates
	rustCargoPreFlight = librarianrust.VerifyCargoTools
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
		return rustPublishCrates(ctx, librarianrust.ToSidekickReleaseConfig(cfg.Release), dryRun, skipSemverChecks, lastTag, files)
	default:
		return fmt.Errorf("publish not implemented for %q", cfg.Language)
	}
}

// verifyRequiredTools verifies all the necessary language-agnostic tools are installed.
func verifyRequiredTools(ctx context.Context, language string, cfg *config.Release) error {
	gitExe := "git"
	if cfg != nil {
		gitExe = command.GetExecutablePath(cfg.Preinstalled, "git")
	}
	if err := git.GitVersion(ctx, gitExe); err != nil {
		return err
	}
	if err := git.GitRemoteURL(ctx, gitExe, cfg.Remote); err != nil {
		return err
	}
	switch language {
	case languageFake:
		return nil
	case languageRust:
		var rustTools []rust.Tool
		for _, t := range cfg.Tools["cargo"] {
			rustTools = append(rustTools, rust.Tool{Name: t.Name, Version: t.Version})
		}
		cargoExe := "cargo"
		if cfg != nil {
			cargoExe = command.GetExecutablePath(cfg.Preinstalled, "cargo")
		}
		if err := rustCargoPreFlight(ctx, cargoExe, rustTools); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown language: %s", language)
	}
	return nil
}
