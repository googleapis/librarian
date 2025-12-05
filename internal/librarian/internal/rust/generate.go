// Copyright 2025 Google LLC
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

package rust

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	googleapisDir, err := sourceDir(ctx, sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	discoveryDir, err := sourceDir(ctx, sources.Discovery, discoveryRepo)
	if err != nil {
		return err
	}
	if len(library.Channels) != 1 {
		return fmt.Errorf("the Rust generator only supports a single channel per library")
	}
	sidekickConfig := toSidekickConfig(library, library.Channels[0], googleapisDir, discoveryDir)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickrust.Generate(model, library.Output, sidekickConfig); err != nil {
		return err
	}
	if err := command.Run("taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	if err := command.Run("cargo", "fmt", "-p", library.Name); err != nil {
		return err
	}
	return nil
}

func sourceDir(ctx context.Context, source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(ctx, repo, source.Commit, source.SHA256)
}

// DefaultLibraryName derives a library name from a channel path.
// For example: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func DefaultLibraryName(channel string) string {
	return strings.ReplaceAll(channel, "/", "-")
}

// DefaultOutput derives an output path from a channel path and default output.
// For example: google/cloud/secretmanager/v1 with default src/generated/
// returns src/generated/cloud/secretmanager/v1.
func DefaultOutput(channel, defaultOutput string) string {
	return filepath.Join(defaultOutput, strings.TrimPrefix(channel, "google/"))
}
