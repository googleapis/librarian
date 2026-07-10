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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

func runPHPMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	libs, err := findPHPLibraries(repoPath)
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
					Name:     "google/gapic-generator-php",
					Version:  "v1.21.2",
					Package:  "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz",
					SHA256:   "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					Build:    []string{"composer install"},
				},
			},
		},
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	log.Printf("Successfully migrated %d PHP libraries configuration skeleton", len(cfg.Libraries))
	return nil
}

var (
	owlbotSourceWithVersionRegexp    = regexp.MustCompile(`^/([a-zA-Z0-9_/]+)/\((v[0-9a-zA-Z]+)\)/.*-php/.*$`)
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

func extractAPIPath(source string) (string, bool) {
	if matches := owlbotSourceWithVersionRegexp.FindStringSubmatch(source); len(matches) == 3 {
		return matches[1] + "/" + matches[2], true
	}
	if matches := owlbotSourceWithoutVersionRegexp.FindStringSubmatch(source); len(matches) == 2 {
		return matches[1], true
	}
	return "", false
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
		if path, ok := extractAPIPath(spec.Source); ok {
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
func findPHPLibraries(repoPath string) ([]*config.Library, error) {
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

		libs = append(libs, &config.Library{
			Name:    name,
			Version: version,
			APIs:    apis,
		})
	}
	return libs, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
