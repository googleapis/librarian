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

// Package swift provides functionality for generating Swift client libraries.
package swift

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickswift "github.com/googleapis/librarian/internal/sidekick/swift"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a Swift client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	if IsMixedLibrary(library) {
		return generateModule(ctx, cfg, library, src)
	}
	if len(library.APIs) != 1 {
		return fmt.Errorf("the Swift generator only supports a single api per library")
	}
	var pc *config.Protoc
	if cfg != nil && cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}
	modelConfig, err := libraryToModelConfig(library, library.APIs[0], src, pc)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return err
	}
	if err = sidekickswift.Generate(ctx, model, library.Output, library, nil); err != nil {
		return err
	}
	if needsRepoMetadata(model, library) {
		repoMetadata, err := createRepoMetadata(cfg, library, src)
		if err != nil {
			return err
		}
		if err := repoMetadata.Write(library.Output); err != nil {
			return err
		}
	}
	return nil
}

// Format formats a generated Swift library.
func Format(ctx context.Context, library *config.Library) error {
	// Collect unique output directories to format to avoid running the formatter
	// multiple times on the same directory (e.g. library.Output and module.Output mapping to same path).
	dirs := map[string]struct{}{}
	if library.Output != "" {
		dirs[library.Output] = struct{}{}
	}
	if library.Swift != nil {
		for _, module := range library.Swift.Modules {
			if module.Output != "" {
				dirs[module.Output] = struct{}{}
			}
		}
	}

	if len(dirs) == 0 {
		return nil
	}

	args := []string{"format", "--in-place", "--recursive"}
	for dir := range dirs {
		args = append(args, dir)
	}
	return command.Run(ctx, "swift-format", args...)
}

// DefaultLibraryName derives a library name from an API path.
//
// Note that what librarian calls "a library" is really a "package" in Swift.
//
// For example: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func DefaultLibraryName(api string) string {
	return strings.ReplaceAll(api, "/", "-")
}

func libraryToModelConfig(library *config.Library, apiCfg *config.API, src *sources.Sources, pc *config.Protoc) (*parser.ModelConfig, error) {
	svcConfig, err := serviceconfig.Find(src.Googleapis, apiCfg.Path, config.LanguageSwift)
	if err != nil {
		return nil, err
	}

	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	if library.Swift != nil && len(library.Swift.IncludeList) > 0 {
		sourceConfig.IncludeList = library.Swift.IncludeList
	}
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	var skippedIDs []string
	if library.Swift != nil && len(library.Swift.SkippedIds) > 0 {
		skippedIDs = library.Swift.SkippedIds
	}

	modelCfg := &parser.ModelConfig{
		Language:            config.LanguageSwift,
		SpecificationFormat: specFormat,
		ServiceConfig:       svcConfig.ServiceConfig,
		SpecificationSource: apiCfg.Path,
		Source:              sourceConfig,
		Protoc:              pc,
		Override: api.ModelOverride{
			SkippedIDs: skippedIDs,
		},
	}
	if library.Swift != nil && library.Swift.Discovery != nil {
		pollers := make([]*api.Poller, len(library.Swift.Discovery.Pollers))
		for i, poller := range library.Swift.Discovery.Pollers {
			pollers[i] = &api.Poller{
				Prefix:   poller.Prefix,
				MethodID: poller.MethodID,
			}
		}
		modelCfg.Discovery = &api.Discovery{
			OperationID: library.Swift.Discovery.OperationID,
			Pollers:     pollers,
		}
	}
	return modelCfg, nil
}
