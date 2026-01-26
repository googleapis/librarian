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
	"slices"
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
	errBothVersionAndAllFlag = errors.New("cannot specify both --version and --all")
	errReleaseCommitNotFound = errors.New("no release commit found")
	errReleaseConfigEmpty    = errors.New("release config not set in librarian.yaml")

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

func bumpCommand() *cli.Command {
	return &cli.Command{
		Name:      "bump",
		Usage:     "update versions and prepare release artifacts",
		UsageText: "librarian bump [library] [--all] [--version=<version>]",
		Description: `bump updates version numbers and prepares the files needed for a new release.

If a library name is given, only that library is updated. The --all flag updates every
library in the workspace. When a library is specified explicitly, the --version flag can
be used to override the new version.

Examples:
  librarian bump <library>           # update version for one library
  librarian bump --all               # update versions for all libraries`,
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
		Action: bumpAction,
	}
}

func bumpAction(ctx context.Context, cmd *cli.Command) error {
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
		return errors.Join(errConfigNotFound, err)
	}
	return runBump(ctx, cfg, all, libraryName, versionOverride)
}

func runBump(ctx context.Context, cfg *config.Config, all bool, libraryName, versionOverride string) error {
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

	if all {
		if err := bumpAll(ctx, cfg, lastTag, gitExe); err != nil {
			return err
		}
	} else {
		lib, err := findLibrary(cfg, libraryName)
		if err != nil {
			return err
		}
		if err := bumpLibrary(ctx, cfg, lib, lastTag, gitExe, versionOverride); err != nil {
			return err
		}
	}

	if err := postBump(ctx, cfg); err != nil {
		return err
	}
	return RunTidyOnConfig(ctx, cfg)
}

func bumpAll(ctx context.Context, cfg *config.Config, lastTag, gitExe string) error {
	filesChanged, err := git.FilesChangedSince(ctx, lastTag, gitExe, cfg.Release.IgnoredChanges)
	if err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		if lib.SkipPublish {
			continue
		}
		output := libraryOutput(cfg.Language, lib, cfg.Default)
		if !hasChangesIn(output, filesChanged) {
			continue
		}
		if err := bumpLibrary(ctx, cfg, lib, lastTag, gitExe, ""); err != nil {
			return err
		}
	}
	return nil
}

func hasChangesIn(dir string, filesChanged []string) bool {
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	for _, f := range filesChanged {
		if strings.HasPrefix(f, dir) {
			return true
		}
	}
	return false
}

func bumpLibrary(ctx context.Context, cfg *config.Config, lib *config.Library, lastTag, gitExe, versionOverride string) error {
	opts := languageVersioningOptions[cfg.Language]
	version, err := deriveNextVersion(ctx, gitExe, cfg, lib, opts, versionOverride)
	if err != nil {
		return err
	}
	output := libraryOutput(cfg.Language, lib, cfg.Default)

	switch cfg.Language {
	case languageFake:
		return fakeBumpLibrary(lib, version)
	case languageRust:
		return rust.Bump(ctx, lib, output, version, gitExe, lastTag)
	default:
		return fmt.Errorf("%q does not support bump", cfg.Language)
	}
}

// postBump performs post version bump cleanup and maintenance tasks after libraries have been processed.
func postBump(ctx context.Context, cfg *config.Config) error {
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

// findLibrary returns a library with the given name from the config.
func findLibrary(c *config.Config, name string) (*config.Library, error) {
	if c.Libraries == nil {
		return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, name)
	}
	for _, library := range c.Libraries {
		if library.Name == name {
			return library, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, name)
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
		if errors.Is(err, ErrLibraryNotFound) {
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
	branchLibCfg, err := findLibrary(branchLibrarianCfg, libName)
	if err != nil {
		return "", err
	}
	return branchLibCfg.Version, nil
}

// findReleasedLibraries determines which libraries are released by the
// change in config from cfgBefore to cfgAfter. This includes libraries
// which exist (with a version) in cfgAfter but either didn't exist or
// didn't have a version in cfgBefore. An error is returned if any version
// transition is a regression (e.g. 1.2.0 to 1.1.0, or 1.2.0 to "").
func findReleasedLibraries(cfgBefore, cfgAfter *config.Config) ([]string, error) {
	results := []string{}
	for _, candidate := range cfgAfter.Libraries {
		candidateBefore, err := findLibrary(cfgBefore, candidate.Name)
		if err != nil {
			// Any error other than "not found" is effectively fatal.
			if !errors.Is(err, ErrLibraryNotFound) {
				return nil, err
			}
			if candidate.Version != "" {
				if err := semver.ValidateNext("", candidate.Version); err != nil {
					return nil, err
				}
				results = append(results, candidate.Name)
			}
			continue
		}
		if candidate.Version == "" {
			if candidateBefore.Version != "" {
				return nil, fmt.Errorf("library %q has no version; was at version %q", candidate.Name, candidateBefore.Version)
			}
			continue
		}
		if candidate.Version == candidateBefore.Version {
			continue
		}
		if err := semver.ValidateNext(candidateBefore.Version, candidate.Version); err != nil {
			return nil, err
		}
		results = append(results, candidate.Name)
	}
	return results, nil
}

// findLatestReleaseCommitHash finds the latest (most recent) commit hash
// which released the library named by libraryName, or which released any libraries
// if libraryName is empty. (See findReleasedLibraries for the definition of what it
// means for a commit to release a library.)
func findLatestReleaseCommitHash(ctx context.Context, gitExe, libraryName string) (string, error) {
	commits, err := git.FindCommitsForPath(ctx, gitExe, librarianConfigPath)
	if err != nil {
		return "", err
	}
	// We're working backwards from HEAD, so we need to keep track of the commit
	// *before* (in iteration order; after in chronological order) the one where
	// we actually spot it's done a release.
	var candidateConfig *config.Config
	candidateCommit := ""
	for _, commit := range commits {
		commitCfgContent, err := git.ShowFileAtRevision(ctx, gitExe, commit, librarianConfigPath)
		if err != nil {
			return "", err
		}
		commitCfg, err := yaml.Unmarshal[config.Config]([]byte(commitCfgContent))
		if err != nil {
			return "", err
		}
		// On the first iteration, we just use the loaded configuration
		// as the candidate to check against in later iterations. For everything
		// else, we see whether the candidate performed a release - and if so,
		// we return that commit.
		if candidateConfig != nil {
			released, err := findReleasedLibraries(commitCfg, candidateConfig)
			if err != nil {
				return "", err
			}
			if len(released) > 0 && (libraryName == "" || slices.Contains(released, libraryName)) {
				return candidateCommit, nil
			}
		}
		candidateConfig = commitCfg
		candidateCommit = commit
	}
	return "", errReleaseCommitNotFound
}
