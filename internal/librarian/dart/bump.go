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
		if strings.Contains(err.Error(), "Package not available") {
			// First release: no breaking changes to compare against, keep old version.
			return lib.Version, lib.Version, nil
		}
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

func maybeBumpLibrary(ctx context.Context, cloudDeps []string, newVersions map[string]string, lib *config.Library, defaults *config.Default) (string, error) {
	if lib.SkipRelease || lib.Version == "" {
		return lib.Version, nil
	}

	oldVersion := lib.Version
	if oldVersion == "" {
		return "", fmt.Errorf("version not set for library %s", lib.Name)
	}

	packageDir := libraryOutput(lib, defaults)

	depsChanged := false
	libraryChanged := false
	var lastReleaseTagCommit string
	if defaults != nil && defaults.TagFormat != "" {
		tagName := git.FormatTagName(defaults.TagFormat, lib.Name, lib.Version)
		commit, err := git.GetCommitHash(ctx, command.Git, tagName)
		if err != nil {
			// If tag doesn't exist yet, we treat it as changed.
			libraryChanged = true
		} else {
			lastReleaseTagCommit = commit
			filesChanged, err := git.FilesChangedSince(ctx, command.Git, lastReleaseTagCommit, []string{})
			if err != nil {
				return "", err
			}
			libraryChanged = git.HasChangesIn(packageDir, "", filesChanged)
		}
	} else {
		// If tag format is not configured, fallback to true.
		libraryChanged = true
	}

	newDeps := make(map[string]string)
	for _, dep := range cloudDeps {
		if v, ok := newVersions[dep]; ok && v != "" {
			depsChanged = true
			newDeps["package:"+dep] = "^" + v
		}
	}

	publishedVersion, neededVersion, err := recommendedVersion(ctx, lib, defaults)
	if err != nil {
		return "", err
	}

	if semver.MaxVersion(oldVersion, neededVersion) == oldVersion {
		// The version has already been incremented to/past what is required.
		return oldVersion, nil
	}

	newVersion := neededVersion
	// If there are no changes to the package then `neededVersion` will be the published
	// version of the package.
	if (depsChanged || libraryChanged) && neededVersion == publishedVersion {
		bumped, err := semver.DeriveNext(semver.Patch, oldVersion, semver.DeriveNextOptions{
			DowngradePreGAChanges: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to derive next version: %w", err)
		}
		newVersion = bumped
	}

	if newVersion != oldVersion {
		pubspecPath := filepath.Join(packageDir, "pubspec.yaml")
		if err := updatePubspecVersion(pubspecPath, newVersion); err != nil {
			return "", err
		}

		var commits []string
		if lastReleaseTagCommit != "" {
			var err error
			commits, err = getCommitsSince(ctx, lastReleaseTagCommit, packageDir)
			if err != nil {
				return "", err
			}
		}

		if len(commits) == 0 {
			if lastReleaseTagCommit == "" {
				commits = []string{"Initial release."}
			} else {
				commits = []string{"Dependency updates."}
			}
		}

		if err := updateChangelog(packageDir, newVersion, commits); err != nil {
			return "", err
		}
	}

	return newVersion, nil
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
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			commits = append(commits, trimmed)
		}
	}
	return commits, nil
}

func updateChangelog(packageDir, version string, commits []string) error {
	changelogPath := filepath.Join(packageDir, "CHANGELOG.md")
	var entry []string
	entry = append(entry, fmt.Sprintf("## %s", version))
	entry = append(entry, "")
	for _, commit := range commits {
		entry = append(entry, fmt.Sprintf("- %s", commit))
	}
	entryStr := strings.Join(entry, "\n") + "\n\n"

	content, err := os.ReadFile(changelogPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// File does not exist, create new
		newContent := "# Changelog\n\n" + entryStr
		return os.WriteFile(changelogPath, []byte(newContent), 0644)
	}

	// File exists, prepend entry after heading
	changelogContent := string(content)
	if strings.HasPrefix(changelogContent, "# Changelog") {
		rest := strings.TrimPrefix(changelogContent, "# Changelog")
		rest = strings.TrimLeft(rest, "\r\n ")
		newContent := "# Changelog\n\n" + entryStr + rest
		return os.WriteFile(changelogPath, []byte(newContent), 0644)
	}

	newContent := entryStr + changelogContent
	return os.WriteFile(changelogPath, []byte(newContent), 0644)
}

// Bump updates the version number and dependencies of Dart packages in the workspace.
func Bump(ctx context.Context, cfg *config.Config, all bool, libraryName, versionOverride string) error {
	if !all {
		return errors.New("bumping a single Dart library not supported, use --all")
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
		newVersions[lib] = newVersion

		if libraryByName[lib].Version != newVersion {
			libraryByName[lib].Version = newVersion

			if cfg.Default != nil && cfg.Default.Dart != nil && cfg.Default.Dart.Packages != nil {
				if _, ok := cfg.Default.Dart.Packages["package:"+lib]; ok {
					cfg.Default.Dart.Packages["package:"+lib] = "^" + newVersion
				}
			}

			for _, other := range sorted {
				if slices.Contains(deps[other], lib) {
					newDeps := map[string]string{"package:" + lib: "^" + newVersion}
					if err := updatePubspecDependencyVersions(libraryByName[other], cfg.Default, newDeps); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
