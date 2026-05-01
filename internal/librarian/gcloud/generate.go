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

// Package gcloud provides librarian wiring for the gcloud-go code generator,
// which emits a self-contained Go CLI module mirroring a gcloud command tree.
package gcloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
	"github.com/googleapis/librarian/internal/sources"
)

// ErrNoProtosFound is returned when no .proto files are found in the API directory.
var ErrNoProtosFound = errors.New("no .proto files found")

// Generate emits a Go CLI module for each API in the library.
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

	modulePath := defaultModulePath(library)
	if library.Gcloud != nil && library.Gcloud.ModulePath != "" {
		modulePath = library.Gcloud.ModulePath
	}

	for _, api := range library.APIs {
		if err := generateAPI(api, googleapisDir, outDir, modulePath); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	return nil
}

func generateAPI(api *config.API, googleapisDir, outDir, modulePath string) error {
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
		"",
		"",
	)
	if err != nil {
		return err
	}
	return sidekickgcloud.Generate(model, outDir, modulePath)
}

func defaultModulePath(library *config.Library) string {
	name := library.Name
	if name == "" {
		name = "gcloud-cli"
	}
	return "example.com/" + name
}

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
		return nil, fmt.Errorf("%w: %q", ErrNoProtosFound, apiDir)
	}
	return protos, nil
}

func findServiceConfig(googleapisDir, apiPath string) (string, error) {
	sc, err := serviceconfig.Find(googleapisDir, apiPath, config.LanguageGcloud)
	if err != nil {
		return "", err
	}
	if sc.ServiceConfig == "" {
		return "", fmt.Errorf("no service config found for api %q", apiPath)
	}
	return filepath.Join(googleapisDir, sc.ServiceConfig), nil
}
