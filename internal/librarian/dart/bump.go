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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

// IgnoredChanges is a list of files that are to be ignored as changes during the bump command.
var IgnoredChanges = []string{
	".repo-metadata.json",
	"docs/README.rst",
}

func getCloudDeps(ctx context.Context, libraries []*config.Library) (map[string][]string, error) {
	output, err := command.Output(ctx, command.Dart, "pub", "deps", "--json")
	if err != nil {
		return nil, err
	}
	var data struct {
		Packages []struct {
			Name         string   `json:"name"`
			Dependencies []string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, fmt.Errorf("failed to parse dart pub deps output: %w", err)
	}

	libNames := make(map[string]bool)
	for _, lib := range libraries {
		libNames[lib.Name] = true
	}

	depsMap := make(map[string][]string)
	for _, pkg := range data.Packages {
		if !libNames[pkg.Name] {
			continue
		}
		var deps []string
		for _, dep := range pkg.Dependencies {
			if libNames[dep] {
				deps = append(deps, dep)
			}
		}
		slices.Sort(deps)
		depsMap[pkg.Name] = deps
	}

	return depsMap, nil
}

func sortLibraries(libraryByName map[string]*config.Library, deps map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for name := range libraryByName {
		pkgDeps := deps[name]
		inDegree[name] = len(pkgDeps)
		for _, dep := range pkgDeps {
			dependents[dep] = append(dependents[dep], name)
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	slices.Sort(queue)

	var sorted []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, dep := range dependents[curr] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		slices.Sort(queue)
	}

	if len(sorted) < len(libraryByName) {
		return nil, fmt.Errorf("cycle detected in dependency DAG")
	}

	return sorted, nil
}

func bumpLibrary(ctx context.Context, cloudDeps []string, newVersions map[string]string, lib *config.Library, defaults *config.Default) (string, error) {
	oldVersion := lib.Version
	if oldVersion == "" {
		return "", fmt.Errorf("version not set for library %s", lib.Name)
	}

	depsChanged := false
	libraryChanged := true
	newDeps := make(map[string]string)
	for _, dep := range cloudDeps {
		if v, ok := newVersions[dep]; ok && v != "" {
			depsChanged = true
			newDeps["package:"+dep] = "^" + v
		}
	}

	packageDir := libraryOutput(lib, defaults)
	reportPath := filepath.Join(os.TempDir(), fmt.Sprintf("report-%s.json", lib.Name))
	defer os.Remove(reportPath)
	pubspecPath := filepath.Join(packageDir, "pubspec.yaml")

	var neededVersion string
	var publishedVersion string
	output, err := command.Output(ctx, "dart-apitool", "diff", "--old", "pub://"+lib.Name, "--new", packageDir,
		"--report-format", "json", "--report-file-path", reportPath, "--version-check-mode", "fully",
		"--no-set-exit-on-version-check-failure")
	if err != nil {
		if strings.Contains(err.Error(), "Package not available") {
			// First release: no breaking changes to compare against, keep old version.
			return oldVersion, nil
		} else {
			return "", fmt.Errorf("dart-apitool failed: %w (output: %s)", err, output)
		}
	} else {
		// Read the report file
		reportContent, err := os.ReadFile(reportPath)
		if err != nil {
			return "", fmt.Errorf("failed to read API report: %w", err)
		}
		var report struct {
			Version struct {
				Needed string `json:"needed"`
				Old    string `json:"old"` // The published version of the package.
			} `json:"version"`
		}
		if err := json.Unmarshal(reportContent, &report); err != nil {
			return "", fmt.Errorf("failed to parse API report: %w", err)
		}
		neededVersion = report.Version.Needed
		if neededVersion == "" {
			return "", fmt.Errorf("API report did not contain recommended version")
		}
		publishedVersion = report.Version.Old
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

	if err := updatePubspecVersion(pubspecPath, newVersion); err != nil {
		return "", err
	}

	return newVersion, nil
}

func libraryOutput(lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	var defaultOut string
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return filepath.Join(defaultOut, lib.Name)
}

func updatePubspecDependencyVersions(lib *config.Library, defaults *config.Default, newDeps map[string]string) error {
	packageDir := libraryOutput(lib, defaults)
	pubspecPath := filepath.Join(packageDir, "pubspec.yaml")
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inDeps := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "dependencies:") {
			inDeps = true
			newLines = append(newLines, line)
			continue
		} else if inDeps && len(line) > 0 && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			inDeps = false
		}
		if inDeps && strings.HasPrefix(line, "  ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				depName := strings.TrimSpace(parts[0])
				if constraint, ok := newDeps["package:"+depName]; ok {
					indent := line[:len(line)-len(trimmed)]
					newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, depName, constraint))
					continue
				}
			}
		}
		newLines = append(newLines, line)
	}
	return os.WriteFile(pubspecPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func updatePubspecVersion(path string, newVersion string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "version:") {
			newLines = append(newLines, fmt.Sprintf("version: %s", newVersion))
			continue
		}
		newLines = append(newLines, line)
	}
	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

// Bump updates the version number and dependencies of Dart packages in the workspace.
func Bump(ctx context.Context, cfg *config.Config, all bool, libraryName, versionOverride string) error {
	libraryByName := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		//		if lib.SkipRelease {
		//			continue
		//		}
		libraryByName[lib.Name] = lib
	}

	fmt.Printf("libraryByName: %+#v\n", libraryByName)

	deps, err := getCloudDeps(ctx, cfg.Libraries)
	if err != nil {
		return err
	}
	fmt.Printf("deps: %+#v\n", deps)

	sorted, err := sortLibraries(libraryByName, deps)
	if err != nil {
		return err
	}
	fmt.Printf("sorted libraries: %v\n", sorted)

	newVersions := make(map[string]string)

	for _, lib := range sorted {
		newVersion, err := bumpLibrary(ctx, deps[lib], newVersions, libraryByName[lib], cfg.Default)
		if err != nil {
			return err
		}
		newVersions[lib] = newVersion
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

	return nil
	/*

		adj := make(map[string][]string)
		inDegree := make(map[string]int)
		for name := range libraryByName {
			inDegree[name] = 0
		}

		for _, lib := range cfg.Libraries {
			if lib.SkipRelease {
				continue
			}
			if lib.Dart == nil || lib.Dart.Packages == nil {
				continue
			}
			for dep := range lib.Dart.Packages {
				depName := strings.TrimPrefix(dep, "package:")
				if _, ok := libraryByName[depName]; ok {
					adj[depName] = append(adj[depName], lib.Name)
					inDegree[lib.Name]++
				}
			}
		}

		initiallyChanged := make(map[string]bool)
		if !all {
			lib, ok := libraryByName[libraryName]
			if !ok {
				return fmt.Errorf("library %s not found in configuration", libraryName)
			}
			initiallyChanged[lib.Name] = true
		} else {
			for _, lib := range cfg.Libraries {
				if lib.SkipRelease || lib.Version == "" {
					continue
				}
				lastReleaseTagName := formatTagName(cfg.Default.TagFormat, lib)
				lastReleaseTagCommit, err := git.GetCommitHash(ctx, gitExe, lastReleaseTagName)
				if err != nil {
					// If tag doesn't exist yet, treat it as changed
					initiallyChanged[lib.Name] = true
					continue
				}
				filesChanged, err := git.FilesChangedSince(ctx, gitExe, lastReleaseTagCommit, IgnoredChanges)
				if err != nil {
					return err
				}
				if libraryChanged(cfg, lib, filesChanged) {
					initiallyChanged[lib.Name] = true
				}
			}
		}

		var queue []string
		for name, deg := range inDegree {
			if deg == 0 {
				queue = append(queue, name)
			}
		}
		slices.Sort(queue)

		newVersions := make(map[string]string)
		versionChanged := make(map[string]bool)
		for _, lib := range cfg.Libraries {
			newVersions[lib.Name] = lib.Version
		}

		processedCount := 0
		for len(queue) > 0 {
			currName := queue[0]
			queue = queue[1:]
			processedCount++

			currLib := libraryByName[currName]
			var depsChanged bool

			// Check if dependencies changed, and update constraints
			if currLib.Dart != nil && currLib.Dart.Packages != nil {
				for dep := range currLib.Dart.Packages {
					depName := strings.TrimPrefix(dep, "package:")
					if versionChanged[depName] {
						depsChanged = true
						currLib.Dart.Packages[dep] = "^" + newVersions[depName]
					}
				}
			}

			// Check if we should bump the version of currLib
			shouldBump := initiallyChanged[currName] || depsChanged
			if shouldBump {
				var nextVer string
				var err error
				if currName == libraryName && versionOverride != "" {
					nextVer, err = deriveNextVersion(currLib, versionOverride)
				} else {
					nextVer, err = deriveNextVersion(currLib, "")
				}
				if err != nil {
					return err
				}
				currLib.Version = nextVer
				newVersions[currName] = nextVer
				versionChanged[currName] = true
			}

			// If the version or dependencies changed, update the pubspec.yaml file
			if shouldBump {
				pubspecPath := filepath.Join(libraryOutput(currLib, cfg.Default), "pubspec.yaml")
				if err := updatePubspecFile(pubspecPath, newVersions[currName], currLib.Dart.Packages); err != nil {
					return err
				}
			}

			for _, dep := range adj[currName] {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
			slices.Sort(queue)
		}

		if processedCount < len(libraryByName) {
			return fmt.Errorf("cycle detected in dependency DAG")
		}

		return nil
	*/
}

/*
func deriveNextVersion(library *config.Library, versionOverride string) (string, error) {
	if versionOverride != "" {
		if err := semver.ValidateNext(library.Version, versionOverride); err != nil {
			return "", err
		}
		return versionOverride, nil
	}
	if library.Version == "" {
		return defaultVersion, nil
	}
	return semver.DeriveNext(semver.Minor, library.Version, semver.DeriveNextOptions{})
}

func formatTagName(tagFormat string, library *config.Library) string {
	tag := strings.ReplaceAll(tagFormat, "{name}", library.Name)
	return strings.ReplaceAll(tag, "{version}", library.Version)
}

func libraryChanged(cfg *config.Config, library *config.Library, filesChanged []string) bool {
	output := libraryOutput(library, cfg.Default)
	if !strings.HasSuffix(output, "/") {
		output += "/"
	}
	for _, f := range filesChanged {
		if strings.HasPrefix(f, output) {
			return true
		}
	}
	return false
}

*/
