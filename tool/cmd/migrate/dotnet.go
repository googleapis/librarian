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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
)

// ApisJSON represents the root of the apis.json file.
type ApisJSON struct {
	APIs          []APIEntry     `json:"apis"`
	PackageGroups []PackageGroup `json:"packageGroups"`
}

// APIEntry represents a single API entry in apis.json.
type APIEntry struct {
	ID            string            `json:"id"`
	Version       string            `json:"version"`
	Type          string            `json:"type"`
	Generator     string            `json:"generator"`
	ProtoPath     string            `json:"protoPath"`
	Transport     string            `json:"transport"`
	Dependencies  map[string]string `json:"dependencies"`
	BlockRelease  string            `json:"blockRelease"`
	NoVersionHist bool              `json:"noVersionHistory"`
}

// PackageGroup represents a package group in apis.json.
type PackageGroup struct {
	ID         string   `json:"id"`
	PackageIDs []string `json:"packageIds"`
}

func runDotnetMigration(ctx context.Context, repoPath string) error {
	apisJSON, err := readApisJSON(repoPath)
	if err != nil {
		return err
	}
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg := buildDotnetConfig(apisJSON, src)
	if cfg == nil {
		return fmt.Errorf("no libraries found to migrate")
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, cfg); err != nil {
		return errTidyFailed
	}
	log.Printf("Successfully migrated %d .NET libraries", len(cfg.Libraries))
	return nil
}

func readApisJSON(repoPath string) (*ApisJSON, error) {
	path := filepath.Join(repoPath, "generator-input", "apis.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading apis.json: %w", err)
	}
	var apisJSON ApisJSON
	if err := json.Unmarshal(data, &apisJSON); err != nil {
		return nil, fmt.Errorf("parsing apis.json: %w", err)
	}
	return &apisJSON, nil
}

func buildDotnetConfig(apisJSON *ApisJSON, src *config.Source) *config.Config {
	// Build a map of API ID to entry for package group lookups.
	apiByID := make(map[string]*APIEntry, len(apisJSON.APIs))
	for i := range apisJSON.APIs {
		apiByID[apisJSON.APIs[i].ID] = &apisJSON.APIs[i]
	}

	var libs []*config.Library
	for _, api := range apisJSON.APIs {
		lib := &config.Library{
			Name:    api.ID,
			Version: api.Version,
		}

		// Set APIs from protoPath for generated libraries.
		isHandwritten := api.Generator == "None"
		if !isHandwritten && api.ProtoPath != "" {
			lib.APIs = []*config.API{
				{Path: api.ProtoPath},
			}
		}

		// Only set transport when it differs from the default "grpc+rest".
		if api.Transport != "" && api.Transport != "grpc+rest" {
			lib.Transport = api.Transport
		}

		// Handwritten libraries are veneers.
		if isHandwritten {
			lib.Veneer = true
		}

		// Set release level for preview versions.
		v := strings.ToLower(api.Version)
		if strings.Contains(v, "alpha") || strings.Contains(v, "beta") {
			lib.ReleaseLevel = "preview"
		}

		if api.BlockRelease != "" {
			lib.SkipRelease = true
		}

		// Build .NET-specific configuration.
		var dotnet *config.DotnetPackage
		if api.Generator == "proto" {
			if dotnet == nil {
				dotnet = &config.DotnetPackage{}
			}
			dotnet.Generator = "proto"
		}

		// Filter dependencies: remove "default" and "project" values.
		if len(api.Dependencies) > 0 {
			filtered := make(map[string]string)
			for k, v := range api.Dependencies {
				if v == "default" || v == "project" {
					continue
				}
				filtered[k] = v
			}
			if len(filtered) > 0 {
				if dotnet == nil {
					dotnet = &config.DotnetPackage{}
				}
				dotnet.Dependencies = filtered
			}
		}

		lib.Dotnet = dotnet
		libs = append(libs, lib)
	}

	// Apply package groups.
	for _, pg := range apisJSON.PackageGroups {
		// Find the first package in the group that is a generated library
		// (has a protoPath).
		for _, pkgID := range pg.PackageIDs {
			api, ok := apiByID[pkgID]
			if !ok || api.ProtoPath == "" {
				continue
			}
			// Find the corresponding library and set the package group.
			for _, lib := range libs {
				if lib.Name != pkgID {
					continue
				}
				if lib.Dotnet == nil {
					lib.Dotnet = &config.DotnetPackage{}
				}
				lib.Dotnet.PackageGroup = pg.PackageIDs
				break
			}
			break
		}
	}

	if len(libs) == 0 {
		return nil
	}

	sort.Slice(libs, func(i, j int) bool {
		return libs[i].Name < libs[j].Name
	})

	return &config.Config{
		Language: config.LanguageDotnet,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			Output:    "apis",
			TagFormat: "{name}-{version}",
		},
		Libraries: libs,
	}
}
