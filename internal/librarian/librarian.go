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
	"github.com/urfave/cli/v3"
)

// ErrLibraryNotFound is returned when the specified library is not found in config.
var ErrLibraryNotFound = errors.New("library not found")

const (
	librarianConfigPath = "librarian.yaml"
	languageDart        = "dart"
	languageFake        = "fake"
	languageGo          = "go"
	languageRust        = "rust"
	languagePython      = "python"
)

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
			tidyCommand(),
			updateCommand(),
			versionCommand(),
			publishCommand(),
			tagCommand(),
		},
	}
	return cmd.Run(ctx, args)
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
