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

// Package gcloud provides gcloud command binary generation for librarian.
package gcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a gcloud command binary for a library.
//
// For each API in the library, it parses the protos and service config via
// the sidekick parser and invokes the sidekick gcloud generator, writing the
// results to the library's output directory.
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

	resolvedSrcs := &sources.Sources{Googleapis: googleapisDir}
	for _, api := range library.APIs {
		if err := generateAPI(api, resolvedSrcs, googleapisDir, outDir); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	return nil
}

func generateAPI(api *config.API, srcs *sources.Sources, googleapisDir, outDir string) error {
	sc, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageGcloud)
	if err != nil {
		return err
	}
	if sc.ServiceConfig == "" {
		return fmt.Errorf("no service config found for api %q", api.Path)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: api.Path,
		ServiceConfig:       sc.ServiceConfig,
		Source: &sources.SourceConfig{
			Sources:     srcs,
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		return err
	}
	return sidekickgcloud.Generate(model, outDir)
}
