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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
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
					Checksum: "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
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

// findPHPLibraries scans the repository root directory for subdirectories containing
// both a VERSION file and a composer.json file. It assumes each matching subdirectory
// represents a PHP library, where the library name is the subdirectory's name and
// the version is extracted from the VERSION file.
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
		libs = append(libs, &config.Library{
			Name:    name,
			Version: version,
		})
	}
	return libs, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
