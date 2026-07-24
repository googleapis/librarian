// Copyright 2026 Google LLC
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

package dart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/semver"
)

// Bump updates the version number and dependencies of Dart packages in the workspace.
//
// The algorithm (minus edge cases) is:
//  1. Find the dependencies of all packages in the repository.
//  2. Order the packages so that no package appears before any of its dependencies.
//  3. For each package in order:
//     a. Check if any files changed since the last release
//     b. Use dart-apitool to see what the recommended next version is.
//     c. If the version should be updated based:
//     - Update the version in pubspec.yaml.
//     - Update CHANGELOG.md.
//     - Update the "dependencies:" section of each dependent package.
//     - Update the version for the package in cfg.Default.Dart.Packages if the package
//     has an entry there.
func Bump(ctx context.Context, cfg *config.Config, all bool, libraryName, versionOverride string) error {
	if !all {
		return errors.New("bumping a single Dart package not supported, use --all")
	}

	libraryByName := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		libraryByName[lib.Name] = lib
	}

	deps, _, err := getDeps(ctx, cfg.Libraries)
	if err != nil {
		return err
	}

	sorted, err := sortByDeps(libraryByName, deps)
	if err != nil {
		return err
	}

	newVersions := make(map[string]string)

	for _, lib := range sorted {
		newVersion, err := maybeBumpLibrary(ctx, deps[lib], newVersions, libraryByName[lib], cfg.Default)
		if err != nil {
			return err
		}

		if libraryByName[lib].Version != newVersion {
			libraryByName[lib].Version = newVersion
			newVersions[lib] = newVersion

			if cfg.Default != nil && cfg.Default.Dart != nil && cfg.Default.Dart.Packages != nil {
				if _, ok := cfg.Default.Dart.Packages["package:"+lib]; ok {
					cfg.Default.Dart.Packages["package:"+lib] = "^" + newVersion
				}
			}

			for _, other := range sorted {
				if slices.Contains(deps[other], lib) {
					newDeps := map[string]string{lib: "^" + newVersion}
					if err := updatePubspecDependencyVersions(libraryByName[other], cfg.Default, newDeps); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func maybeBumpLibrary(ctx context.Context, cloudDeps []string, newVersions map[string]string, lib *config.Library, defaults *config.Default) (string, error) {
	if lib.SkipRelease || lib.Version == "" {
		return lib.Version, nil
	}

	packageDir := libraryOutput(lib, defaults)

	publishedVersion, err := getPublishedVersion(ctx, lib.Name)
	if err != nil {
		return "", err
	}

	if publishedVersion == "" {
		// The packaged is unpublished so the only work to do is create a CHANGELOG.md.
		if err := updateChangelog(ctx, packageDir, lib.Version, "", false); err != nil {
			return "", err
		}
		return lib.Version, nil
	}

	if lib.Version != publishedVersion {
		return "", fmt.Errorf("pub.dev version %s does not match librarian.yaml version %s", publishedVersion, lib.Version)
	}

	libraryChanged := false
	var lastReleaseTagCommit string
	if defaults == nil || defaults.TagFormat == "" {
		return "", errors.New("no tag format configured")
	}

	tagName := git.FormatTagName(defaults.TagFormat, lib.Name, lib.Version)
	commit, err := git.GetCommitHash(ctx, command.Git, tagName)
	if err != nil {
		return "", err
	}
	lastReleaseTagCommit = commit
	filesChanged, err := git.FilesChangedSince(ctx, command.Git, lastReleaseTagCommit, []string{})
	if err != nil {
		return "", err
	}
	libraryChanged = git.HasChangesIn(packageDir, "", filesChanged)

	depsChanged := false
	for _, dep := range cloudDeps {
		if _, ok := newVersions[dep]; ok {
			depsChanged = true
			break
		}
	}

	publishedVersion, neededVersion, err := recommendedVersion(ctx, lib, defaults)
	if err != nil {
		return "", err
	}

	if neededVersion == lib.Version && !depsChanged && !libraryChanged {
		return lib.Version, nil
	}

	newVersion := neededVersion
	// If there are no changes to the package then `neededVersion` will be the published
	// version of the package.
	if (depsChanged || libraryChanged) && neededVersion == publishedVersion {
		bumped, err := semver.DeriveNext(semver.Patch, lib.Version, semver.DeriveNextOptions{
			DowngradePreGAChanges: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to derive next version: %w", err)
		}
		newVersion = bumped
	}

	if newVersion != lib.Version {
		pubspecPath := filepath.Join(packageDir, "pubspec.yaml")
		if err := updatePubspecVersion(pubspecPath, newVersion); err != nil {
			return "", err
		}

		if err := updateChangelog(ctx, packageDir, newVersion, lastReleaseTagCommit, depsChanged); err != nil {
			return "", err
		}
	}

	return newVersion, nil
}

func recommendedVersion(ctx context.Context, lib *config.Library, defaults *config.Default) (string, string, error) {
	packageDir := libraryOutput(lib, defaults)
	reportPath := filepath.Join(os.TempDir(), fmt.Sprintf("report-%s.json", lib.Name))
	defer os.Remove(reportPath)

	var neededVersion string
	var publishedVersion string
	output, err := command.Output(ctx, command.DartAPITool, "diff", "--old", "pub://"+lib.Name, "--new", packageDir,
		"--report-format", "json", "--report-file-path", reportPath, "--version-check-mode", "fully",
		"--no-set-exit-on-version-check-failure")
	if err != nil {
		return "", "", fmt.Errorf("dart-apitool failed: %w (output: %s)", err, output)
	}

	// Read the report file
	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read API report: %w", err)
	}
	var report struct {
		Version struct {
			Needed string `json:"needed"`
			Old    string `json:"old"` // The published version of the package.
		} `json:"version"`
	}
	if err := json.Unmarshal(reportContent, &report); err != nil {
		return "", "", fmt.Errorf("failed to parse API report: %w", err)
	}
	neededVersion = report.Version.Needed
	if neededVersion == "" {
		return "", "", fmt.Errorf("API report did not contain recommended version")
	}
	publishedVersion = report.Version.Old

	return publishedVersion, neededVersion, nil
}

func getCommitsSince(ctx context.Context, lastReleaseTagCommit, packageDir string) ([]string, error) {
	if lastReleaseTagCommit == "" {
		return nil, nil
	}
	output, err := command.Output(ctx, command.Git, "log", fmt.Sprintf("%s..HEAD", lastReleaseTagCommit), "--format=%s", "--", packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits since %s for %s: %w", lastReleaseTagCommit, packageDir, err)
	}
	var commits []string
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			commits = append(commits, trimmed)
		}
	}
	return commits, nil
}

func updateChangelog(ctx context.Context, packageDir, version, lastReleaseTagCommit string, depsChanged bool) error {
	changelogPath := filepath.Join(packageDir, "CHANGELOG.md")

	if lastReleaseTagCommit == "" {
		return os.WriteFile(changelogPath, fmt.Appendf(nil, `# Changelog

## %s

- initial release

`, version), 0644)
	}

	changes, err := getCommitsSince(ctx, lastReleaseTagCommit, packageDir)
	if err != nil {
		return err
	}
	if depsChanged {
		changes = append(changes, "chore: update cloud dependencies")
	}

	if len(changes) == 0 {
		return fmt.Errorf("updating changelog for unchanged package: %s", packageDir)
	}

	var entry []string
	entry = append(entry, fmt.Sprintf("## %s", version))
	entry = append(entry, "")
	for _, commit := range changes {
		entry = append(entry, fmt.Sprintf("- %s", commit))
	}
	entryStr := strings.Join(entry, "\n") + "\n\n"

	content, err := os.ReadFile(changelogPath)
	newTopOfFile := "# Changelog\n\n" + entryStr
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File does not exist, create new
		return os.WriteFile(changelogPath, []byte(newTopOfFile), 0644)
	}

	rest := string(content)
	if after, ok := strings.CutPrefix(rest, "# Changelog"); ok {
		rest = after
		rest = strings.TrimLeft(rest, "\r\n ")
	}

	return os.WriteFile(changelogPath, []byte(newTopOfFile+rest), 0644)
}
