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
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sidekick/rust_prost"
)

const (
	googleapisRepo  = "github.com/googleapis/googleapis"
	discoveryRepo   = "github.com/googleapis/discovery-artifact-manager"
	protobufRepo    = "github.com/protocolbuffers/protobuf"
	conformanceRepo = "github.com/googleapis/conformance-tests"
	showcaseRepo    = "github.com/googleapis/gapic-showcase"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	dirs, err := getSourceDirs(ctx, sources)
	if err != nil {
		return err
	}

	googleapisDir := dirs["googleapis"]

	if library.Veneer {
		return generateVeneer(ctx, library, googleapisDir)
	}

	if len(library.Channels) != 1 {
		return fmt.Errorf("the Rust generator only supports a single channel per library")
	}

	sidekickConfig := toSidekickConfig(library, library.Channels[0], googleapisDir,
		dirs["discovery"], dirs["protobuf-src"], dirs["conformance"], dirs["showcase"])
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickrust.Generate(ctx, model, library.Output, sidekickConfig); err != nil {
		return err
	}
	return nil
}

func getSourceDirs(ctx context.Context, sources *config.Sources) (map[string]string, error) {
	dirs := make(map[string]string)
	sourceMap := map[string]struct {
		cfg  *config.Source
		repo string
	}{
		"googleapis":   {sources.Googleapis, googleapisRepo},
		"discovery":    {sources.Discovery, discoveryRepo},
		"protobuf-src": {sources.ProtobufSrc, protobufRepo},
		"conformance":  {sources.Conformance, conformanceRepo},
		"showcase":     {sources.Showcase, showcaseRepo},
	}
	for name, info := range sourceMap {
		dir, err := sourceDir(ctx, info.cfg, info.repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get source dir for %s: %w", name, err)
		}
		dirs[name] = dir
	}
	return dirs, nil
}

// Format formats a generated Rust library. Must be called sequentially;
// parallel calls cause race conditions as cargo fmt runs cargo metadata,
// which competes for locks on the workspace Cargo.toml and Cargo.lock.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "fmt", "-p", library.Name); err != nil {
		return err
	}
	return nil
}

func generateVeneer(ctx context.Context, library *config.Library, googleapisDir string) error {
	if library.Rust == nil || len(library.Rust.Modules) == 0 {
		return fmt.Errorf("veneer %q has no modules defined", library.Name)
	}
	for _, module := range library.Rust.Modules {
		sidekickConfig := moduleToSidekickConfig(library, module, googleapisDir)
		model, err := parser.CreateModel(sidekickConfig)
		if err != nil {
			return fmt.Errorf("module %s: %w", module.Output, err)
		}
		switch sidekickConfig.General.Language {
		case "rust":
			err = sidekickrust.Generate(ctx, model, module.Output, sidekickConfig)
		case "rust+prost":
			err = rust_prost.Generate(ctx, model, module.Output, sidekickConfig)
		default:
			err = fmt.Errorf("unknown language: %s", sidekickConfig.General.Language)
		}
		if err != nil {
			return fmt.Errorf("module %s: %w", module.Output, err)
		}
	}
	return nil
}

// Keep returns the list of files to preserve when cleaning the output directory.
func Keep(library *config.Library) ([]string, error) {
	if !library.Veneer {
		return append(library.Keep, "Cargo.toml"), nil
	}

	// For veneers, keep all files outside module output directories. We walk
	// library.Output and keep files not under any module.Output.
	var keep []string
	moduleOutputs := make(map[string]bool)
	for _, m := range library.Rust.Modules {
		moduleOutputs[m.Output] = true
	}
	err := filepath.WalkDir(library.Output, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if moduleOutputs[path] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(library.Output, path)
		if err != nil {
			return err
		}
		keep = append(keep, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keep, nil
}

func sourceDir(ctx context.Context, source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(ctx, repo, source.Commit, source.SHA256, source.Subpath)
}

// DefaultLibraryName derives a library name from a channel path.
// For example: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func DefaultLibraryName(channel string) string {
	return strings.ReplaceAll(channel, "/", "-")
}

// DeriveChannelPath derives a channel path from a library name.
// For example: google-cloud-secretmanager-v1 -> google/cloud/secretmanager/v1.
func DeriveChannelPath(name string) string {
	return strings.ReplaceAll(name, "-", "/")
}

// DefaultOutput derives an output path from a channel path and default output.
// For example: google/cloud/secretmanager/v1 with default src/generated/
// returns src/generated/cloud/secretmanager/v1.
func DefaultOutput(channel, defaultOutput string) string {
	return filepath.Join(defaultOutput, strings.TrimPrefix(channel, "google/"))
}
