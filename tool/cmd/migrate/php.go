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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

var protoMappings = map[string]string{
	"//google/cloud/location:location_proto": "google/cloud/location/locations.proto",
	"//google/iam/v1:iam_policy_proto":       "google/iam/v1/iam_policy.proto",
}

func runPHPMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	libs, err := findPHPLibraries(repoPath, src.Dir)
	if err != nil {
		return err
	}
	cfg := &config.Config{
		Language: config.LanguagePhp,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Libraries: libs,
		Tools: &config.Tools{
			Composer: []*config.ComposerTool{
				{
					Name:    "google/gapic-generator-php",
					Version: "v1.21.2",
					Package: "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz",
					SHA256:  "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					Build:   []string{"composer install"},
				},
			},
			Protoc: &config.Protoc{
				Version: "31.0",
				SHA256:  "24e2ed32060b7c990d5eb00d642fde04869d7f77c6d443f609353f097799dd42",
			},
		},
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	log.Printf("Successfully migrated %d PHP libraries configuration to librarian.yaml", len(cfg.Libraries))
	return nil
}

var (
	owlbotSourceWithVersionRegexp    = regexp.MustCompile(`^/([a-zA-Z0-9_/]+)/\((v[0-9a-zA-Z|]+)\)/.*-php/.*$`)
	owlbotSourceWithoutVersionRegexp = regexp.MustCompile(`^/([a-zA-Z0-9_/]+)/.*-php/.*$`)
)

type owlBotConfig struct {
	DeepCopyRegex []deepCopyRegexSpec `yaml:"deep-copy-regex"`
	APIName       string              `yaml:"api-name"`
}

type deepCopyRegexSpec struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

// extractAPIPaths extracts target API paths from an OwlBot source matcher pattern.
// It supports both unversioned paths and versioned paths, including union matchers
// (e.g. "(v1|v1beta2)") which are expanded into separate versioned paths.
// Returns nil if the pattern is invalid.
func extractAPIPaths(source string) []string {
	if matches := owlbotSourceWithVersionRegexp.FindStringSubmatch(source); len(matches) == 3 {
		// matches[1] is the base path (e.g. "google/cloud/secretmanager")
		// matches[2] is the version or version union (e.g. "v1" or "v1|v1beta2")
		base := matches[1]
		versions := strings.Split(matches[2], "|")
		var paths []string
		for _, v := range versions {
			paths = append(paths, base+"/"+v)
		}
		return paths
	}
	if matches := owlbotSourceWithoutVersionRegexp.FindStringSubmatch(source); len(matches) == 2 {
		// matches[1] is the full path without a version suffix (e.g. "google/identity/accesscontextmanager/type")
		return []string{matches[1]}
	}
	return nil
}

func extractAPIsFromOwlBot(owlbotPath string) ([]*config.API, error) {
	if !fileExists(owlbotPath) {
		return nil, nil
	}
	owlbot, err := yaml.Read[owlBotConfig](owlbotPath)
	if err != nil {
		return nil, err
	}
	var apis []*config.API
	seenAPIs := make(map[string]bool)
	for _, spec := range owlbot.DeepCopyRegex {
		for _, path := range extractAPIPaths(spec.Source) {
			if !seenAPIs[path] {
				seenAPIs[path] = true
				apis = append(apis, &config.API{Path: path})
			}
		}
	}
	return apis, nil
}

// findPHPLibraries scans the repository root directory for subdirectories containing
// both a VERSION file and a composer.json file. It assumes each matching subdirectory
// represents a PHP library, where the library name is the subdirectory's name and
// the version is extracted from the VERSION file.
// It also attempts to parse .OwlBot.yaml to extract API paths.
func findPHPLibraries(repoPath string, googleapisDir string) ([]*config.Library, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	var libs []*config.Library
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		versionPath := filepath.Join(repoPath, name, "VERSION")
		composerPath := filepath.Join(repoPath, name, "composer.json")
		// Check if both VERSION and composer.json exist
		if !fileExists(versionPath) || !fileExists(composerPath) {
			continue
		}
		versionBytes, err := os.ReadFile(versionPath)
		if err != nil {
			return nil, fmt.Errorf("reading version for %s: %w", name, err)
		}
		version := strings.TrimSpace(string(versionBytes))

		apis, err := extractAPIsFromOwlBot(filepath.Join(repoPath, name, ".OwlBot.yaml"))
		if err != nil {
			return nil, fmt.Errorf("extracting APIs from OwlBot config for %s: %w", name, err)
		}

		for _, api := range apis {
			additionalProtos, err := parsePHPBazel(googleapisDir, api.Path)
			if err != nil {
				log.Printf("Warning: failed to parse BUILD.bazel for %s: %v", api.Path, err)
				continue
			}
			if len(additionalProtos) > 0 {
				api.PHP = &config.PHPAPI{
					AdditionalProtos: additionalProtos,
				}
			}
		}

		libs = append(libs, &config.Library{
			Name:    name,
			Version: version,
			APIs:    apis,
		})
	}
	return libs, nil
}

func parsePHPBazel(googleapisDir, apiPath string) ([]string, error) {
	file, err := parseBazel(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	var additionalProtos []string
	if rules := file.Rules("proto_library_with_info"); len(rules) > 0 {
		rule := rules[0]
		if attr := rule.Attr("deps"); attr != nil {
			for _, dep := range extractStrings(attr) {
				// Ignore local targets within the same package.
				if strings.HasPrefix(dep, ":") {
					continue
				}
				// Ignore common resources which are handled natively.
				// TODO(https://github.com/googleapis/librarian/issues/6813):
				// load this to dedicated config
				if strings.Contains(dep, "common_resources_proto") {
					continue
				}
				// Ignore LROs since PHP does not compile LRO methods as mixins.
				if strings.HasPrefix(dep, "//google/longrunning:") {
					continue
				}
				// Ignore policy_proto as it only defines structs; the IAMPolicy service is in iam_policy_proto.
				if dep == "//google/iam/v1:policy_proto" {
					continue
				}
				if protoPath, ok := protoMappings[dep]; ok {
					additionalProtos = append(additionalProtos, protoPath)
				} else {
					log.Printf("Warning: unmapped dependency %q found in %s/BUILD.bazel", dep, apiPath)
				}
			}
		}
	}
	return additionalProtos, nil
}

func parseBazel(googleapisDir, dir string) (*build.File, error) {
	path := filepath.Join(googleapisDir, dir, "BUILD.bazel")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	file, err := build.ParseBuild(path, data)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// extractStrings returns all string literals found within a Bazel expression.
func extractStrings(expr build.Expr) []string {
	var res []string
	build.Walk(expr, func(e build.Expr, _ []build.Expr) {
		if s, ok := e.(*build.StringExpr); ok {
			res = append(res, s.Value)
		}
	})
	return res
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
