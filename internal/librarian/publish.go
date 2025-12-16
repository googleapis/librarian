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

	"github.com/googleapis/librarian/internal/change"
	"github.com/googleapis/librarian/internal/config"
	rust "github.com/googleapis/librarian/internal/librarian/internal/rust"
	rustrelease "github.com/googleapis/librarian/internal/sidekick/rust_release"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	rustPublishCrates  = rustrelease.PublishCrates
	rustCargoPreFlight = rustrelease.CargoPreFlight
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
	if err := preflight(ctx, cfg.Language, cfg.Release); err != nil {
		return err
	}
	if err := change.AssertGitStatusClean(ctx, cfg.Release.GetExecutablePath("git")); err != nil {
		return err
	}
	lastTag, err := change.GetLastTag(ctx, cfg.Release.GetExecutablePath("git"), cfg.Release.Remote, cfg.Release.Branch)
	if err != nil {
		return err
	}
	files, err := change.FilesChangedSince(ctx, lastTag, cfg.Release.GetExecutablePath("git"), cfg.Release.IgnoredChanges)
	if err != nil {
		return err
	}
	switch cfg.Language {
	case "rust":
		return rustPublishCrates(ctx, rust.ToSidekickReleaseConfig(cfg.Release), dryRun, skipSemverChecks, lastTag, files)
	default:
		return fmt.Errorf("publish not implemented for %q", cfg.Language)
	}
}

// preflight verifies all the necessary language-agnostic tools are installed.
func preflight(ctx context.Context, language string, cfg *config.Release) error {
	if err := change.GitVersion(ctx, cfg.GetExecutablePath("git")); err != nil {
		return err
	}
	if err := change.GitRemoteURL(ctx, cfg.GetExecutablePath("git"), cfg.Remote); err != nil {
		return err
	}
	switch language {
	case "rust":
		if err := rustCargoPreFlight(ctx, rust.ToSidekickReleaseConfig(cfg)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown language: %s", language)
	}
	return nil
}
