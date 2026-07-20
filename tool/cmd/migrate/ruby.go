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
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	versionedAPIPath = regexp.MustCompile(`^/(.+/(v\d+\w*))/(.+)-ruby/(.*)$`)
)

type owlbotYaml struct {
	DeepCopyRegex []owlbotSrc `yaml:"deep-copy-regex"`
}

type owlbotSrc struct {
	Source string `yaml:"source"`
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
	libs, err := findRubyLibraries(repoPath)
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

func findRubyLibraries(repoPath string) ([]*config.Library, error) {
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
	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Name < libraries[j].Name
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
			if strings.HasPrefix(other.Name, prefix) {
				wrapperOf = append(wrapperOf, other.Name)
			} else {
				// Since libraries are sorted by name, the wrapped libraries
				// must be consecutive.
				break
			}
		}
		if len(wrapperOf) > 0 {
			lib.Ruby = &config.RubyPackage{
				WrapperOf: wrapperOf,
			}
		}
	}
}
