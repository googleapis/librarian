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
	"github.com/googleapis/librarian/internal/librarian/dart"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/librarian/swift"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:      "publish",
		Hidden:    true,
		Usage:     "publish client libraries",
		UsageText: "librarian publish",
		Description: `publish releases the libraries that were updated in a release commit
prepared by librarian bump.

Only Dart, Rust, and Swift are supported.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print commands without executing",
			},
			&cli.BoolFlag{
				Name:  "dry-run-keep-going",
				Usage: "print commands without executing, don't stop on error",
			},
			&cli.BoolFlag{
				Name:  "skip-semver-checks",
				Usage: "skip semantic versioning checks",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "streams output of publishing commands executed",
			},
			&cli.StringFlag{
				Name:  "remote-url-format",
				Usage: "template for remote repository URLs (e.g. 'git@github.com:googleapis/{name}.git')",
			},
			&cli.StringFlag{
				Name:  "origin",
				Value: "HEAD",
				Usage: "source commit or branch to split from",
			},
			&cli.StringFlag{
				Name:  "remote-branch",
				Value: "main",
				Usage: "branch name on the remote repository",
			},
			&cli.StringFlag{
				Name:  "upstream",
				Value: config.RemoteUpstream,
				Usage: "name of the upstream git remote",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil {
				return err
			}
			if cfg.Language == config.LanguageRust {
				return rustPublish(ctx, cfg, cmd)
			}
			if cfg.Language == config.LanguageDart {
				return dartPublish(ctx, cfg, cmd)
			}
			if cfg.Language == config.LanguageSwift {
				return swiftPublish(ctx, cfg, cmd)
			}
			return fmt.Errorf("publish is not supported for %q", cfg.Language)
		},
	}
}

func dartPublish(ctx context.Context, cfg *config.Config, cmd *cli.Command) error {
	dryRun := cmd.Bool("dry-run")
	skipSemverChecks := cmd.Bool("skip-semver-checks")
	dryRunKeepGoing := cmd.Bool("dry-run-keep-going")
	verbose := cmd.Bool("verbose")
	command.Verbose = verbose
	setupLogger(verbose)
	return dart.Publish(ctx, dart.PublishParams{
		Config:           cfg,
		DryRun:           dryRun,
		DryRunKeepGoing:  dryRunKeepGoing,
		SkipSemverChecks: skipSemverChecks,
		Verbose:          verbose,
	})
}

func rustPublish(ctx context.Context, cfg *config.Config, cmd *cli.Command) error {
	dryRun := cmd.Bool("dry-run")
	skipSemverChecks := cmd.Bool("skip-semver-checks")
	dryRunKeepGoing := cmd.Bool("dry-run-keep-going")
	verbose := cmd.Bool("verbose")
	command.Verbose = verbose
	setupLogger(verbose)
	return rust.Publish(ctx, rust.PublishParams{
		Config:           cfg,
		DryRun:           dryRun,
		DryRunKeepGoing:  dryRunKeepGoing,
		SkipSemverChecks: skipSemverChecks,
		Verbose:          verbose,
		IgnoredChanges:   IgnoredChanges,
	})
}

func swiftPublish(ctx context.Context, cfg *config.Config, cmd *cli.Command) error {
	dryRun := cmd.Bool("dry-run")
	skipSemverChecks := cmd.Bool("skip-semver-checks")
	dryRunKeepGoing := cmd.Bool("dry-run-keep-going")
	verbose := cmd.Bool("verbose")
	remoteURLFormat := cmd.String("remote-url-format")
	origin := cmd.String("origin")
	remoteBranch := cmd.String("remote-branch")
	upstream := cmd.String("upstream")
	command.Verbose = verbose
	setupLogger(verbose)
	return swift.Publish(ctx, swift.PublishParams{
		Config:           cfg,
		Libraries:        cmd.Args().Slice(),
		DryRun:           dryRun,
		DryRunKeepGoing:  dryRunKeepGoing,
		SkipSemverChecks: skipSemverChecks,
		Verbose:          verbose,
		RemoteURLFormat:  remoteURLFormat,
		Origin:           origin,
		RemoteBranch:     remoteBranch,
		Upstream:         upstream,
	})
}
