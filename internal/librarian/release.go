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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errLibraryNotFound    = errors.New("library not found")
	errReleaseConfigEmpty = errors.New("librarian Release.Config field empty")
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
	gitExe := "git"
	if cfg.Release != nil {
		gitExe = command.GetExecutablePath(cfg.Release.Preinstalled, "git")
	}
	if err := git.AssertGitStatusClean(ctx, gitExe); err != nil {
		return err
	}

	if cfg.Release == nil {
		return errReleaseConfigEmpty
	}
	lastTag, err := git.GetLastTag(ctx, gitExe, cfg.Release.Remote, cfg.Release.Branch)
	if err != nil {
		return err
	}

	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return errNoGoogleapiSourceInfo
	}
	googleapisDir, err := fetchSource(ctx, cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}

	if all {
		if err = releaseAll(ctx, cfg, lastTag, gitExe, googleapisDir); err != nil {
			return err
		}
	} else {
		libConfg, err := libraryByName(cfg, libraryName)
		if err != nil {
			return err
		}
		_, err = prepareLibrary(cfg.Language, libConfg, cfg.Default, "", false)
		if err != nil {
			return err
		}
		if err = releaseLibrary(ctx, cfg, libConfg, libConfg.Output, lastTag, gitExe, googleapisDir); err != nil {
			return err
		}
	}
	return RunTidyOnConfig(ctx, cfg)
}

func releaseAll(ctx context.Context, cfg *config.Config, lastTag, gitExe, googleapisDir string) error {
	filesChanged, err := git.FilesChangedSince(ctx, lastTag, gitExe, cfg.Release.IgnoredChanges)
	if err != nil {
		return err
	}
	for _, library := range cfg.Libraries {
		_, err := prepareLibrary(cfg.Language, library, cfg.Default, "", false)
		if err != nil {
			return err
		}
		if shouldRelease(library, filesChanged, library.Output) {
			if err := releaseLibrary(ctx, cfg, library, library.Output, lastTag, gitExe, googleapisDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldRelease(library *config.Library, filesChanged []string, srcPath string) bool {
	if library.SkipPublish {
		return false
	}
	pathWithTrailingSlash := srcPath
	if !strings.HasSuffix(pathWithTrailingSlash, "/") {
		pathWithTrailingSlash = pathWithTrailingSlash + "/"
	}
	for _, path := range filesChanged {
		if strings.Contains(path, pathWithTrailingSlash) {
			return true
		}
	}
	return false
}

func releaseLibrary(ctx context.Context, cfg *config.Config, libConfig *config.Library, srcPath, lastTag, gitExe, googleapisDir string) error {
	switch cfg.Language {
	case languageFake:
		return fakeReleaseLibrary(libConfig)
	case languageRust:
		release, err := rust.ManifestVersionNeedsBump(gitExe, lastTag, srcPath+"/Cargo.toml")
		if err != nil {
			return err
		}
		if !release {
			return nil
		}
		if err := rust.ReleaseLibrary(libConfig, srcPath); err != nil {
			return err
		}
		copyConfig, err := cloneConfig(cfg)
		if err != nil {
			return err
		}
		if _, err := generateLibrary(ctx, copyConfig, googleapisDir, libConfig.Name); err != nil {
			return err
		}
		if err := formatLibrary(ctx, cfg.Language, libConfig); err != nil {
			return err
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

func cloneConfig(orig *config.Config) (*config.Config, error) {
	data, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}
	var copy config.Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}
