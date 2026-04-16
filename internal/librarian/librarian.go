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

// Package librarian provides functionality for onboarding, generating and
// releasing Google Cloud client libraries.
package librarian

import (
	"context"
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/nodejs"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

// ErrLibraryNotFound is returned when the specified library is not found in config.
var ErrLibraryNotFound = errors.New("library not found")

// Run executes the librarian command with the given arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:      "librarian",
		Usage:     "manage Google Cloud client libraries",
		UsageText: "librarian [command]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "enable verbose logging",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			command.Verbose = cmd.Bool("verbose")
			return ctx, nil
		},
		Commands: []*cli.Command{
			addCommand(),
			generateCommand(),
			bumpCommand(),
			installCommand(),
			tidyCommand(),
			updateCommand(),
			versionCommand(),
			publishCommand(),
			tagCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

func installCommand() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "install tool dependencies for a language",
		UsageText: "librarian install [language]",
		Description: `Install tool dependencies for the given language.
If no language is provided, the language is determined
from librarian.yaml in the current directory.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			lang := cmd.Args().First()
			cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
			if err != nil && lang == "" {
				return err
			}
			if lang == "" {
				lang = cfg.Language
			}
			var tools *config.Tools
			if cfg != nil {
				tools = cfg.Tools
			}

			switch lang {
			case config.LanguageFake:
				return nil
			case config.LanguageGo:
				if err := golang.Install(ctx, tools); err != nil {
					return fmt.Errorf("install go dependencies: %w", err)
				}
				return nil
			case config.LanguageNodejs:
				return nodejs.Install(ctx)
			case config.LanguagePython:
				return python.Install(ctx)
			case config.LanguageRust:
				return rust.Install(ctx, tools)
			default:
				return fmt.Errorf("language %q does not support install", lang)
			}
		},
	}
}

// versionCommand prints the version information.
func versionCommand() *cli.Command {
	return &cli.Command{
		Name:      "version",
		Usage:     "print the version",
		UsageText: "librarian version",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("librarian version %s\n", Version())
			return nil
		},
	}
}
