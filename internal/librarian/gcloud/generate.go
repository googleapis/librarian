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
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a gcloud command binary for a library.
//
// It parses every API in the library via the sidekick parser into a
// model, then invokes the sidekick gcloud generator once with the full
// set so the resulting cmd/gcloud/main.go and internal/generated/<name>
// surface packages live in a single tree under the library's output
// directory.
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
	models := make([]*api.API, 0, len(library.APIs))
	for _, a := range library.APIs {
		model, err := buildModel(a, resolvedSrcs, googleapisDir)
		if err != nil {
			return fmt.Errorf("failed to build api model %q: %w", a.Path, err)
		}
		models = append(models, model)
	}
	return sidekickgcloud.Generate(models, outDir)
}

func buildModel(a *config.API, srcs *sources.Sources, googleapisDir string) (*api.API, error) {
	sc, err := serviceconfig.Find(googleapisDir, a.Path, config.LanguageGcloud)
	if err != nil {
		return nil, err
	}
	if sc.ServiceConfig == "" {
		return nil, fmt.Errorf("no service config found for api %q", a.Path)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: a.Path,
		ServiceConfig:       sc.ServiceConfig,
		Source: &sources.SourceConfig{
			Sources:     srcs,
			ActiveRoots: []string{"googleapis"},
		},
		Codec: map[string]string{
			"copyright-year": "2026",
		},
	}
	return parser.CreateModel(cfg)
}
