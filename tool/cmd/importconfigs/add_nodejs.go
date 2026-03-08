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

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

// owlBotYAML represents the fields needed from an .OwlBot.yaml file.
type owlBotYAML struct {
	DeepCopyRegex []struct {
		Source string `yaml:"source"`
	} `yaml:"deep-copy-regex"`
}

// owlBotSourceRegex extracts the base API path from an .OwlBot.yaml
// deep-copy-regex source pattern. The pattern is always of the form:
// /some/path/(version-regex)/.*-nodejs
var owlBotSourceRegex = regexp.MustCompile(`^/(.+?)/\(`)

func addNodejsCommand() *cli.Command {
	return &cli.Command{
		Name:  "add-nodejs",
		Usage: "add nodejs to sdk.yaml languages for APIs found in google-cloud-node",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "node-repo",
				Usage:    "path to google-cloud-node repo",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "googleapis",
				Usage:    "path to googleapis dir",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			nodeRepo := cmd.String("node-repo")
			googleapisDir := cmd.String("googleapis")
			return runAddNodejs("internal/serviceconfig/sdk.yaml", nodeRepo, googleapisDir)
		},
	}
}

func runAddNodejs(sdkYAMLPath, nodeRepo, googleapisDir string) error {
	apiPaths, err := findNodejsAPIPaths(nodeRepo, googleapisDir)
	if err != nil {
		return fmt.Errorf("finding Node.js API paths: %w", err)
	}
	slog.Info("found Node.js API paths", "count", len(apiPaths))

	apis, err := yaml.Read[[]serviceconfig.API](sdkYAMLPath)
	if err != nil {
		return fmt.Errorf("reading sdk.yaml: %w", err)
	}

	// Build a map from path to index for quick lookup.
	pathIndex := make(map[string]int, len(*apis))
	for i, api := range *apis {
		pathIndex[api.Path] = i
	}

	var added, updated int
	for _, p := range apiPaths {
		if idx, ok := pathIndex[p]; ok {
			// Entry exists. Add "nodejs" if not already present.
			if slices.Contains((*apis)[idx].Languages, "nodejs") {
				continue
			}
			(*apis)[idx].Languages = append((*apis)[idx].Languages, "nodejs")
			sort.Strings((*apis)[idx].Languages)
			updated++
			continue
		}
		// Entry does not exist. Only add if it does not start with "google/cloud/"
		// since those are implicitly allowed.
		if strings.HasPrefix(p, "google/cloud/") {
			continue
		}
		*apis = append(*apis, serviceconfig.API{
			Path:      p,
			Languages: []string{"nodejs"},
		})
		pathIndex[p] = len(*apis) - 1
		added++
	}

	sort.Slice(*apis, func(i, j int) bool {
		return (*apis)[i].Path < (*apis)[j].Path
	})

	if err := yaml.Write(sdkYAMLPath, *apis); err != nil {
		return fmt.Errorf("writing sdk.yaml: %w", err)
	}
	slog.Info("updated sdk.yaml", "added", added, "updated", updated, "total_nodejs_apis", len(apiPaths))
	return nil
}

// findNodejsAPIPaths walks the google-cloud-node packages directory and
// returns all unique API paths (e.g., "google/cloud/speech/v1") that have a
// nodejs_gapic_library rule in their BUILD.bazel.
func findNodejsAPIPaths(nodeRepo, googleapisDir string) ([]string, error) {
	packagesDir := filepath.Join(nodeRepo, "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("reading packages dir: %w", err)
	}

	seen := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgDir := filepath.Join(packagesDir, entry.Name())
		owlBotPath := filepath.Join(pkgDir, ".OwlBot.yaml")
		if _, statErr := os.Stat(owlBotPath); statErr != nil {
			continue
		}
		paths, err := apiPathsFromOwlBot(owlBotPath, googleapisDir)
		if err != nil {
			slog.Warn("skipping package", "package", entry.Name(), "error", err)
			continue
		}
		for _, p := range paths {
			seen[p] = true
		}
	}

	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	sort.Strings(result)
	return result, nil
}

// apiPathsFromOwlBot reads an .OwlBot.yaml file and returns the API paths
// that have a nodejs_gapic_library rule in their BUILD.bazel in googleapis.
func apiPathsFromOwlBot(owlBotPath, googleapisDir string) ([]string, error) {
	owlBot, err := yaml.Read[owlBotYAML](owlBotPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", owlBotPath, err)
	}
	if len(owlBot.DeepCopyRegex) == 0 {
		return nil, nil
	}
	source := owlBot.DeepCopyRegex[0].Source
	matches := owlBotSourceRegex.FindStringSubmatch(source)
	if len(matches) < 2 {
		return nil, fmt.Errorf("cannot parse API path from source: %q", source)
	}
	basePath := matches[1]

	dir := filepath.Join(googleapisDir, basePath)
	versionEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading googleapis directory %s: %w", dir, err)
	}

	var paths []string
	for _, ve := range versionEntries {
		if !ve.IsDir() {
			continue
		}
		if !strings.HasPrefix(ve.Name(), "v") {
			continue
		}
		apiPath := filepath.Join(basePath, ve.Name())
		has, err := hasNodejsGapicLibrary(googleapisDir, apiPath)
		if err != nil {
			return nil, err
		}
		if has {
			paths = append(paths, apiPath)
		}
	}
	return paths, nil
}

// hasNodejsGapicLibrary checks whether the BUILD.bazel file at the given API
// path in googleapis contains a nodejs_gapic_library rule.
func hasNodejsGapicLibrary(googleapisDir, apiPath string) (bool, error) {
	buildPath := filepath.Join(googleapisDir, apiPath, "BUILD.bazel")
	data, err := os.ReadFile(buildPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	file, err := build.ParseBuild(buildPath, data)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %w", buildPath, err)
	}
	return len(file.Rules("nodejs_gapic_library")) > 0, nil
}
