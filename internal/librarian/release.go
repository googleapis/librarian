// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errExecuteNotImplemented   = errors.New("--execute not yet implemented")
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "update versions and prepare release artifacts",
		UsageText: "librarian release [library] [--all] [--execute]",
		Description: `Release updates version numbers and prepares the files needed for a new release.
Without --execute, the command prints the planned changes but does not modify the repository.

With --execute, the command writes updated files, creates tags, and pushes them.

If a library name is given, only that library is updated. The --all flag updates every
library in the workspace.

Examples:
  librarian release <library>           # show planned changes for one library
  librarian release --all               # show planned changes for all libraries
  librarian release <library> --execute # apply changes and tag the release
  librarian release --all --execute`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "update all libraries in the workspace",
			},
			&cli.BoolFlag{
				Name:  "execute",
				Usage: "apply updates, create tags, and push them (default is dry-run)",
			},
		},
		Action: runRelease,
	}
}

func runRelease(ctx context.Context, cmd *cli.Command) error {
	all := cmd.Bool("all")
	execute := cmd.Bool("execute")
	libraryName := cmd.Args().First()
	if !all && libraryName == "" {
		return errMissingLibraryOrAllFlag
	}
	if all && libraryName != "" {
		return errBothLibraryAndAllFlag
	}

	if execute {
		// TODO(https://github.com/googleapis/librarian/issues/2966): implement
		// tagging
		return errExecuteNotImplemented
	}

	cfg, err := config.Read(librarianConfigPath)
	if err != nil {
		return err
	}

	r, ok := language.Releasers[cfg.Language]
	if !ok {
		return fmt.Errorf("release not implemented for language: %s", cfg.Language)
	}

	if all {
		cfg, err = r.ReleaseAll(cfg)
	} else {
		cfg, err = r.ReleaseLibrary(cfg, libraryName)
	}
	if err != nil {
		return err
	}

	return cfg.Write(librarianConfigPath)
}
