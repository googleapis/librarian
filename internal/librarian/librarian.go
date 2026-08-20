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
	"log/slog"
	"os"

	"github.com/googleapis/librarian/internal/command"
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
			setupLogger(command.Verbose)
			return ctx, nil
		},
		Commands: []*cli.Command{
			configCommand(),
			addCommand(),
			generateCommand(),
			bumpCommand(),
			installCommand(),
			tidyCommand(),
			updateCommand(),
			publishCommand(),
			tagCommand(),
			versionCommand(),
			debugCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

// setupLogger configures the default slog logger.
// It uses a text handler writing to stderr at LevelWarn and above by default.
// If verbose is true, the log level is set to LevelDebug.
// Source information (file name and line number) is included in each log entry.
func setupLogger(verbose bool) {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})))
}
