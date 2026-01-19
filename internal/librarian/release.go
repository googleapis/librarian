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
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	defaultPreviewBranch  = "preview"
	defaultMainBranch     = "main"
	defaultVersion        = "0.1.0"
	defaultPreviewVersion = "0.1.0-preview.1"
	zeroVersion           = "0.0.0"
)

var (
	errLibraryNotFound       = errors.New("library not found")
	errReleaseConfigEmpty    = errors.New("librarian Release.Config field empty")
	errBothVersionAndAllFlag = errors.New("cannot specify both --version and --all flag")

	// languageVersioningOptions contains language-specific SemVer versioning
	// options. Over time, languages should align on versioning semantics and
	// this should be removed. If a language does not have specific needs, a
	// default [semver.DeriveNextOptions] is returned for default semantics.
	languageVersioningOptions = map[string]semver.DeriveNextOptions{
		"rust": {
			BumpVersionCore:       true,
			DowngradePreGAChanges: true,
		},
	}
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "update versions and prepare release artifacts",
		UsageText: "librarian release [library] [--all] [--version=<version>]",
		Description: `Release updates version numbers and prepares the files needed for a new release.

If a library name is given, only that library is updated. The --all flag updates every
library in the workspace. When a library is specified explicitly, the --version flag can
be used to override the new version.

Examples:
  librarian release <library>           # update version for one library
  librarian release --all               # update versions for all libraries`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "update all libraries in the workspace",
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "specific version to update to; not valid with --all",
			},
		},
		Action: runRelease,
	}
}

func runRelease(ctx context.Context, cmd *cli.Command) error {
	all := cmd.Bool("all")
	libraryName := cmd.Args().First()
	versionOverride := cmd.String("version")
	if !all && libraryName == "" {
		return errMissingLibraryOrAllFlag
	}
	if all && libraryName != "" {
		return errBothLibraryAndAllFlag
	}
	if all && versionOverride != "" {
		return errBothVersionAndAllFlag
	}
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return errors.Join(errNoYaml, err)
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
	var rustSources *rust.Sources
	if cfg.Language == languageRust {
		rustSources, err = fetchRustSources(ctx, cfg.Sources)
		if err != nil {
			return err
		}
		rustSources.Googleapis = googleapisDir
	}

	if all {
		if err = releaseAll(ctx, cfg, lastTag, gitExe, googleapisDir, rustSources); err != nil {
			return err
		}
	} else {
		libConfg, err := libraryByName(cfg, libraryName)
		if err != nil {
			return err
		}
		_, err = prepareLibrary(cfg.Language, libConfg, cfg.Default, false)
		if err != nil {
			return err
		}
		if err = releaseLibrary(ctx, cfg, libConfg, lastTag, gitExe, versionOverride, googleapisDir, rustSources); err != nil {
			return err
		}
	}

	if err := postRelease(ctx, cfg); err != nil {
		return err
	}
	return RunTidyOnConfig(ctx, cfg)
}

func releaseAll(ctx context.Context, cfg *config.Config, lastTag, gitExe string, googleapisDir string, rustSources *rust.Sources) error {
	filesChanged, err := git.FilesChangedSince(ctx, lastTag, gitExe, cfg.Release.IgnoredChanges)
	if err != nil {
		return err
	}
	for _, library := range cfg.Libraries {
		_, err := prepareLibrary(cfg.Language, library, cfg.Default, false)
		if err != nil {
			return err
		}
		if shouldRelease(library, filesChanged) {
			if err := releaseLibrary(ctx, cfg, library, lastTag, gitExe, "", googleapisDir, rustSources); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldRelease(library *config.Library, filesChanged []string) bool {
	if library.SkipPublish {
		return false
	}
	pathWithTrailingSlash := library.Output
	if !strings.HasSuffix(pathWithTrailingSlash, "/") {
		pathWithTrailingSlash = pathWithTrailingSlash + "/"
	}
	for _, path := range filesChanged {
		if strings.HasPrefix(path, pathWithTrailingSlash) {
			return true
		}
	}
	return false
}

func releaseLibrary(ctx context.Context, cfg *config.Config, libConfig *config.Library, lastTag, gitExe, versionOverride, googleapisDir string, rustSources *rust.Sources) error {
	// If the language doesn't have bespoke versioning options, a default
	// [semver.DeriveNextOptions] instance is returned.
	opts := languageVersioningOptions[cfg.Language]
	nextVersion, err := deriveNextVersion(ctx, gitExe, cfg, libConfig, opts, versionOverride)
	if err != nil {
		return err
	}

	switch cfg.Language {
	case languageFake:
		return fakeReleaseLibrary(libConfig, nextVersion)
	case languageRust:
		release, err := rust.ManifestVersionNeedsBump(gitExe, lastTag, libConfig.Output+"/Cargo.toml")
		if err != nil {
			return err
		}
		if !release {
			return nil
		}
		if err := rust.ReleaseLibrary(libConfig, nextVersion); err != nil {
			return err
		}
		if _, err := generateLibrary(ctx, cfg, libConfig.Name, googleapisDir, rustSources); err != nil {
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

// postRelease performs post-release cleanup and maintenance tasks after libraries have been processed.
func postRelease(ctx context.Context, cfg *config.Config) error {
	switch cfg.Language {
	case languageRust:
		cargoExe := "cargo"
		if cfg.Release != nil {
			cargoExe = command.GetExecutablePath(cfg.Release.Preinstalled, "cargo")
		}
		if err := command.Run(ctx, cargoExe, "update", "--workspace"); err != nil {
			return err
		}
	}
	return nil
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

func deriveNextVersion(ctx context.Context, gitExe string, cfg *config.Config, libConfig *config.Library, opts semver.DeriveNextOptions, versionOverride string) (string, error) {
	// If a version override has been specified, use it - but
	// check that it's not a regression or a no-op.
	if versionOverride != "" {
		if err := semver.ValidateNext(libConfig.Version, versionOverride); err != nil {
			return "", err
		}
		return versionOverride, nil
	}

	// First release, use the appropriate default starting version.
	if libConfig.Version == "" {
		if cfg.Release.Branch == defaultPreviewBranch {
			return defaultPreviewVersion, nil
		}
		return defaultVersion, nil
	}

	if cfg.Release.Branch == defaultPreviewBranch {
		stableVersion, err := loadBranchLibraryVersion(ctx, gitExe, cfg.Release.Remote, defaultMainBranch, libConfig.Name)
		if errors.Is(err, errLibraryNotFound) {
			// If the preview setup precedes the stable setup, ensure stable is always behind.
			stableVersion = zeroVersion
		} else if err != nil {
			return "", err
		}
		return semver.DeriveNextPreview(libConfig.Version, stableVersion, opts)
	}

	return semver.DeriveNext(semver.Minor, libConfig.Version, opts)
}

func loadBranchLibraryVersion(ctx context.Context, gitExe, remote, branch, libName string) (string, error) {
	branchLibrarianCfgFile, err := git.ShowFileAtRemoteBranch(ctx, gitExe, remote, branch, librarianConfigPath)
	if err != nil {
		return "", err
	}
	branchLibrarianCfg, err := yaml.Unmarshal[config.Config]([]byte(branchLibrarianCfgFile))
	if err != nil {
		return "", err
	}
	branchLibCfg, err := libraryByName(branchLibrarianCfg, libName)
	if err != nil {
		return "", err
	}
	return branchLibCfg.Version, nil
}
