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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errLibraryNotFound = errors.New("library not found")

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
	if all {
		cfg, err = releaseAll(cfg)
	} else {
		libConfg, err := libraryByName(cfg, libraryName)
		if err != nil {
			return err
		}
		cfg, err = releaseLibrary(cfg, libConfg)
	}
	if err != nil {
		return err
	}
	return yaml.Write(librarianConfigPath, cfg)
}

func releaseAll(cfg *config.Config) (*config.Config, error) {
	for _, library := range cfg.Libraries {
		if _, err := releaseLibrary(cfg, library); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func releaseLibrary(cfg *config.Config, libConfig *config.Library) (*config.Config, error) {
	switch cfg.Language {
	case "testhelper":
		return testReleaseLibrary(cfg, libConfig)
	case "rust":
		return rust.ReleaseLibrary(cfg, libConfig)
	default:
		return nil, fmt.Errorf("language not supported for release: %q", cfg.Language)
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
