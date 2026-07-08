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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/semver"
)

const defaultVersion = "0.1.0"

// IgnoredChanges is a list of files that are to be ignored as changes during the bump command.
var IgnoredChanges = []string{
	".repo-metadata.json",
	"docs/README.rst",
}

// Bump updates the version number and dependencies of Dart packages in the workspace.
func Bump(ctx context.Context, cfg *config.Config, all bool, libraryName, versionOverride string, gitExe string) error {
	libraryByName := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		if lib.SkipRelease {
			continue
		}
		libraryByName[lib.Name] = lib
	}

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
}

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

func updatePubspecFile(path string, newVersion string, newDeps map[string]string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inDeps := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "version:") {
			newLines = append(newLines, fmt.Sprintf("version: %s", newVersion))
			continue
		}
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
	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}
