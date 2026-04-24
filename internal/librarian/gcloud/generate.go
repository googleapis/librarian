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

// Package gcloud provides gcloud-specific functionality for librarian.
//
// Unlike the surfer tool, which reads its command surface configuration from a
// gcloud.yaml file, this package drives generation directly from the APIs
// declared in librarian.yaml and the protos and service configs that live
// beside them in googleapis.
package gcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
	"github.com/googleapis/librarian/internal/sources"
)

// baseModule is the default Python base module path for generated gcloud
// command groups.
const baseModule = "googlecloudsdk"

// Generate generates gcloud command YAML files for a library.
//
// It parses the protos and service config for each API in the library, builds
// a gcloud command tree, and writes the resulting YAML to the library's
// output directory.
func Generate(ctx context.Context, library *config.Library, srcs *sources.Sources) error {
	googleapisDir, err := filepath.Abs(srcs.Googleapis)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, api := range library.APIs {
		if err := generateAPI(api, googleapisDir, outDir); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	return nil
}

func generateAPI(api *config.API, googleapisDir, outDir string) error {
	protos, err := collectProtos(googleapisDir, api.Path)
	if err != nil {
		return err
	}
	serviceConfigPath, err := findServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return err
	}

	model, err := provider.CreateAPIModel(
		googleapisDir,
		strings.Join(protos, ","),
		serviceConfigPath,
		"", // descriptorFiles
		"", // descriptorFilesToGenerate
	)
	if err != nil {
		return err
	}
	return sidekickgcloud.Generate(model, &provider.Config{}, outDir, baseModule)
}

// collectProtos returns proto file paths under apiPath, relative to
// googleapisDir, using forward slashes so they can be matched against the
// parser include list regardless of platform.
func collectProtos(googleapisDir, apiPath string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory %q: %w", apiDir, err)
	}
	var protos []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".proto" {
			continue
		}
		protos = append(protos, filepath.ToSlash(filepath.Join(apiPath, entry.Name())))
	}
	if len(protos) == 0 {
		return nil, fmt.Errorf("no .proto files found in %q", apiDir)
	}
	return protos, nil
}

// findServiceConfig returns the absolute path to the service config YAML for
// apiPath.
func findServiceConfig(googleapisDir, apiPath string) (string, error) {
	sc, err := serviceconfig.Find(googleapisDir, apiPath, config.LanguageGcloud)
	if err != nil {
		return "", err
	}
	if sc == nil || sc.ServiceConfig == "" {
		return "", fmt.Errorf("no service config found for api %q", apiPath)
	}
	return filepath.Join(googleapisDir, sc.ServiceConfig), nil
}
