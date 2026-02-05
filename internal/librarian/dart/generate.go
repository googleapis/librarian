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

// Package dart provides functionality for generating and releasing Dart client
// libraries.
package dart

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	sidekickdart "github.com/googleapis/librarian/internal/sidekick/dart"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

// Generate generates a Dart client library.
func Generate(ctx context.Context, library *config.Library, sources *source.Sources) error {
	sidekickConfig, err := toSidekickConfig(library, library.APIs[0], sources)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickdart.Generate(ctx, model, library.Output, sidekickConfig); err != nil {
		return err
	}
	return nil
}

// Format formats a generated Dart library.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "dart", "format", library.Output); err != nil {
		return err
	}
	return nil
}

// DefaultOutput returns the default output directory for a Dart library.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}

// DefaultLibraryName derives a library name from an api path.
// For example: google/cloud/secretmanager/v1 -> google_cloud_secretmanager_v1.
func DefaultLibraryName(api string) string {
	api = strings.TrimPrefix(api, "google/cloud/")
	api = strings.TrimPrefix(api, "google/")
	return "google_cloud_" + strings.ReplaceAll(api, "/", "_")
}
