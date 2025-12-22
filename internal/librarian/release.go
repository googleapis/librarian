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
	"errors"
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/githelpers"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errLibraryNotFound = errors.New("library not found")

var (
	rustReleaseLibrary       = rust.ReleaseLibrary
	librarianGenerateLibrary = generateLibrary
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
		},
		Action: runRelease,
	}
}

func runRelease(ctx context.Context, cmd *cli.Command) error {
	all := cmd.Bool("all")
	libraryName := cmd.Args().First()
	if !all && libraryName == "" {
		return errMissingLibraryOrAllFlag
	}
	if all && libraryName != "" {
		return errBothLibraryAndAllFlag
	}
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	gitExe := cfg.Release.GetExecutablePath("git")
	if err := githelpers.AssertGitStatusClean(ctx, gitExe); err != nil {
		return err
	}
	if all {
		err = releaseAll(ctx, cfg)
	} else {
		libConfg, err := libraryByName(cfg, libraryName)
		if err != nil {
			return err
		}
		err = releaseLibrary(ctx, cfg, libConfg)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return yaml.Write(librarianConfigPath, cfg)
}

func releaseAll(ctx context.Context, cfg *config.Config) error {
	for _, library := range cfg.Libraries {
		if err := releaseLibrary(ctx, cfg, library); err != nil {
			return err
		}
	}
	return nil
}

func releaseLibrary(ctx context.Context, cfg *config.Config, libConfig *config.Library) error {

	switch cfg.Language {
	case "testhelper":
		return testReleaseLibrary(libConfig)
	case "rust":
		path := rust.DeriveSrcPath(libConfig, cfg)
		release, err := shouldReleaseLibrary(ctx, cfg, path)
		if err != nil {
			return err
		}
		if release {
			if err := rustReleaseLibrary(cfg, libConfig); err != nil {
				return err
			}
			/*if _, err := librarianGenerateLibrary(ctx, cfg, libConfig.Name); err != nil {
				return err
			}*/
		}
		return nil
	default:
		return fmt.Errorf("language not supported for release: %q", cfg.Language)
	}
}

// libraryByName returns a library with the given name from the config.
func libraryByName(c *config.Config, name string) (*config.Library, error) {
	if c.Libraries == nil {
		return nil, errLibraryNotFound
	}
	for _, library := range c.Libraries {
		if library.Name == name {
			return library, nil
		}
	}
	return nil, errLibraryNotFound
}

func shouldReleaseLibrary(ctx context.Context, cfg *config.Config, path string) (bool, error) {

	gitExe := cfg.Release.GetExecutablePath("git")
	lastTag, err := githelpers.GetLastTag(ctx, gitExe, cfg.Release.Remote, cfg.Release.Branch)
	if err != nil {
		return false, err
	}
	numberOfChanges, err := githelpers.NumberOfChangesInDirectorySinceLastRelease(ctx, gitExe, lastTag, path)
	if err != nil {
		return false, err
	}
	slog.Info("got number of changes ", "num changes", numberOfChanges, "path", path)
	return numberOfChanges > 0, nil
}
