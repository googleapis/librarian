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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	versionedAPIPath = regexp.MustCompile(`^/(.+/(v\d+\w*))/(.+)-ruby/(.*)$`)
	// Skip these directories when searching for libraries.
	skippedDirs = []string{".github"}
)

type owlbotYaml struct {
	DeepCopyRegex []owlbotSrc `yaml:"deep-copy-regex"`
}

type owlbotSrc struct {
	Source string `yaml:"source"`
}

// VersionedBuild represents build configuration parsed from BUILD.bazel for a Ruby API version.
type VersionedBuild struct {
	EnvPrefix       string
	ExtraDeps       string
	PathOverride    string
	ServiceOverride string
}

func runRubyMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg := &config.Config{
		Language: config.LanguageRuby,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Tools: &config.Tools{
			Gem: []*config.GemTool{
				{
					Name:    "gapic-generator-cloud",
					Version: "0.49.0",
				},
				{
					Name:    "grpc",
					Version: "1.78.1",
				},
			},
			Protoc: &config.Protoc{
				Version: "33.2",
				SHA256:  "b24b53f87c151bfd48b112fe4c3a6e6574e5198874f38036aff41df3456b8caf",
			},
		},
	}
	libs, err := findRubyLibraries(src.Dir, repoPath)
	if err != nil {
		return err
	}
	cfg.Libraries = libs
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	log.Printf("Successfully migrated Ruby libraries configuration")
	return nil
}

func findRubyLibraries(googleapisPath, repoPath string) ([]*config.Library, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("reading repository directory: %w", err)
	}
	var libraries []*config.Library
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if slices.Contains(skippedDirs, name) {
			continue
		}
		owlBotPath := filepath.Join(repoPath, name, ".OwlBot.yaml")
		if _, err := os.Stat(owlBotPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("checking OwlBot config: %w", err)
		}
		lib := &config.Library{
			Name: name,
		}
		api, err := parseAPIFromOwlBot(owlBotPath)
		if err != nil {
			return nil, err
		}
		if api != "" {
			lib.APIs = []*config.API{
				{
					Path: api,
				},
			}
			vb, err := parseVersionedBuild(googleapisPath, api)
			if err != nil {
				return nil, err
			}
			if vb != nil {
				lib.APIs[0].Ruby = &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						EnvPrefix:         vb.EnvPrefix,
						ExtraDependencies: vb.ExtraDeps,
						PathOverride:      vb.PathOverride,
						ServiceOverride:   vb.ServiceOverride,
					},
				}
			}
		}
		libraries = append(libraries, lib)
	}
	parseWrapperOf(libraries)
	return libraries, nil
}

func parseAPIFromOwlBot(owlBotPath string) (string, error) {
	data, err := os.ReadFile(owlBotPath)
	if err != nil {
		return "", fmt.Errorf("reading OwlBot config %s: %w", owlBotPath, err)
	}
	owlbot, err := yaml.Unmarshal[owlbotYaml](data)
	if err != nil {
		return "", fmt.Errorf("parsing OwlBot config %s: %w", owlBotPath, err)
	}
	// Skip .github/.Owlbot.yaml.
	if len(owlbot.DeepCopyRegex) == 0 {
		return "", nil
	}
	// We only need the first entry since wrapper library will
	// have different parsing logic.
	src := owlbot.DeepCopyRegex[0].Source
	matches := versionedAPIPath.FindStringSubmatch(src)
	if len(matches) != 5 {
		// A wrapper library doesn't have versioned API path.
		return "", nil
	}
	return matches[1], nil
}

// parseWrapperOf sets the WrapperOf field for wrapper libraries.
func parseWrapperOf(libraries []*config.Library) {
	slices.SortFunc(libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})
	for i, lib := range libraries {
		if len(lib.APIs) != 0 {
			// Skip non-wrapper libraries.
			continue
		}
		var wrapperOf []string
		prefix := lib.Name + "-"
		// Since libraries are sorted by name, the wrapped libraries
		// are guaranteed to appear after the wrapper library.
		for j := i + 1; j < len(libraries); j++ {
			other := libraries[j]
			if !strings.HasPrefix(other.Name, prefix) {
				// Since libraries are sorted by name, the wrapped libraries
				// must be consecutive.
				break
			}
			suffix := strings.TrimPrefix(other.Name, prefix)
			// Verify that the suffix after the prefix represents a valid version,
			// e.g., starting with v followed by a digit.
			// We use simple, string comparison because the migration tool will
			// be removed after language onboarding.
			if len(suffix) > 1 && suffix[0] == 'v' && suffix[1] >= '0' && suffix[1] <= '9' {
				wrapperOf = append(wrapperOf, other.Name)
			}
		}
		if len(wrapperOf) > 0 {
			lib.Ruby = &config.RubyPackage{
				WrapperOf: wrapperOf,
			}
		}
	}
}

func parseVersionedBuild(googleapisDir, apiPath string) (*VersionedBuild, error) {
	file, err := parseBazel(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	vb := &VersionedBuild{}
	if rules := file.Rules("ruby_cloud_gapic_library"); len(rules) > 0 {
		rule := rules[0]
		if attr := rule.Attr("extra_protoc_parameters"); attr != nil {
			for _, dep := range extractStrings(attr) {
				switch {
				case strings.HasPrefix(dep, "ruby-cloud-env-prefix="):
					vb.EnvPrefix, _ = strings.CutPrefix(dep, "ruby-cloud-env-prefix=")
				case strings.HasPrefix(dep, "ruby-cloud-extra-dependencies="):
					vb.ExtraDeps, _ = strings.CutPrefix(dep, "ruby-cloud-extra-dependencies=")
				case strings.HasPrefix(dep, "ruby-cloud-path-override="):
					vb.PathOverride, _ = strings.CutPrefix(dep, "ruby-cloud-path-override=")
				case strings.HasPrefix(dep, "ruby-cloud-service-override="):
					vb.ServiceOverride, _ = strings.CutPrefix(dep, "ruby-cloud-service-override=")
				}
			}
		}
	}
	return vb, nil
}
