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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
)

// DotnetAPIsJSON represents the root of the apis.json file.
type DotnetAPIsJSON struct {
	APIs          []DotnetAPIEntry     `json:"apis"`
	PackageGroups []DotnetPackageGroup `json:"packageGroups"`
}

// DotnetAPIEntry represents a single API entry in apis.json.
type DotnetAPIEntry struct {
	ID           string            `json:"id"`
	Version      string            `json:"version"`
	Generator    string            `json:"generator"`
	ProtoPath    string            `json:"protoPath"`
	Transport    string            `json:"transport"`
	Dependencies map[string]string `json:"dependencies"`
	BlockRelease string            `json:"blockRelease"`
}

// DotnetPackageGroup represents a package group in apis.json.
type DotnetPackageGroup struct {
	ID         string   `json:"id"`
	PackageIDs []string `json:"packageIds"`
}

func runDotnetMigration(ctx context.Context, repoPath string) error {
	apisJSON, err := readDotnetAPIsJSON(repoPath)
	if err != nil {
		return err
	}
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg, err := buildDotnetConfig(apisJSON, src)
	if err != nil {
		return err
	}
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return errTidyFailed
	}
	log.Printf("Successfully migrated %d .NET libraries", len(cfg.Libraries))
	return nil
}

func readDotnetAPIsJSON(repoPath string) (*DotnetAPIsJSON, error) {
	path := filepath.Join(repoPath, "generator-input", "apis.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading apis.json: %w", err)
	}
	var apisJSON DotnetAPIsJSON
	if err := json.Unmarshal(data, &apisJSON); err != nil {
		return nil, fmt.Errorf("parsing apis.json: %w", err)
	}
	return &apisJSON, nil
}

func buildDotnetConfig(apisJSON *DotnetAPIsJSON, src *config.Source) (*config.Config, error) {
	apiByID := make(map[string]*DotnetAPIEntry, len(apisJSON.APIs))
	for i := range apisJSON.APIs {
		apiByID[apisJSON.APIs[i].ID] = &apisJSON.APIs[i]
	}

	var libs []*config.Library
	for _, api := range apisJSON.APIs {
		lib := &config.Library{
			Name:    api.ID,
			Version: api.Version,
		}

		isHandwritten := api.Generator == "None"
		if !isHandwritten && api.ProtoPath == "" {
			return nil, fmt.Errorf("generated library %s has no protoPath", api.ID)
		}
		if !isHandwritten {
			lib.APIs = []*config.API{
				{Path: api.ProtoPath},
			}
		}

		if api.BlockRelease != "" {
			lib.SkipRelease = true
		}

		var dotnet *config.DotnetPackage
		if api.Generator == "proto" {
			dotnet = &config.DotnetPackage{Generator: "proto"}
		}

		if len(api.Dependencies) > 0 {
			if dotnet == nil {
				dotnet = &config.DotnetPackage{}
			}
			dotnet.Dependencies = api.Dependencies
		}

		lib.Dotnet = dotnet
		libs = append(libs, lib)
	}

	if len(libs) == 0 {
		return nil, fmt.Errorf("no libraries found to migrate")
	}

	libByName := make(map[string]*config.Library, len(libs))
	for _, lib := range libs {
		libByName[lib.Name] = lib
	}

	for _, pg := range apisJSON.PackageGroups {
		for _, pkgID := range pg.PackageIDs {
			api, ok := apiByID[pkgID]
			if !ok || api.ProtoPath == "" {
				continue
			}
			lib := libByName[pkgID]
			if lib.Dotnet == nil {
				lib.Dotnet = &config.DotnetPackage{}
			}
			lib.Dotnet.PackageGroup = pg.PackageIDs
			break
		}
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
	}, nil
}
