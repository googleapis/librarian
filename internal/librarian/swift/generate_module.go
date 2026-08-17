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

package swift

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickswift "github.com/googleapis/librarian/internal/sidekick/swift"
	"github.com/googleapis/librarian/internal/sources"
)

func generateModule(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	var pc *config.Protoc
	if cfg != nil && cfg.Tools != nil {
		pc = cfg.Tools.Protoc
	}
	for _, module := range library.Swift.Modules {
		switch module.ModuleType {
		case "package-version":
			if err := sidekickswift.GenerateVersion(ctx, module.Output, library); err != nil {
				return err
			}
		case "swift-protobuf":
			if err := compileProtobufs(ctx, pc, library, module, src); err != nil {
				return err
			}
		case "convert-swift":
			modelConfig := moduleToModelConfig(library, module, src, pc)
			model, err := parser.CreateModel(modelConfig)
			if err != nil {
				return err
			}
			if err := sidekickswift.GenerateConversions(ctx, model, module.Output, library, module); err != nil {
				return err
			}
		case "storage":
			if err := generateSwiftStorage(ctx, pc, library, module, src); err != nil {
				return err
			}
		case "", "default", "grpc-client":
			modelConfig := moduleToModelConfig(library, module, src, pc)
			model, err := parser.CreateModel(modelConfig)
			if err != nil {
				return err
			}
			if err := sidekickswift.Generate(ctx, model, module.Output, library, module); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown module type %q", module.ModuleType)
		}
	}
	return nil
}

func moduleToModelConfig(library *config.Library, module *config.SwiftModule, src *sources.Sources, pc *config.Protoc) *parser.ModelConfig {
	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	// Prefer the module-specific include list if configured, allowing per-module filtering
	// (e.g., selecting only specific dependency `.proto` files for conversion generation).
	// Fall back to the library-wide include list if the module does not define its own.
	if len(module.IncludeList) > 0 {
		sourceConfig.IncludeList = module.IncludeList
	} else if library.Swift != nil && len(library.Swift.IncludeList) > 0 {
		sourceConfig.IncludeList = library.Swift.IncludeList
	}

	var includedIDs []string
	if len(module.IncludedIDs) > 0 {
		includedIDs = module.IncludedIDs
	} else if library.Swift != nil && len(library.Swift.IncludedIDs) > 0 {
		includedIDs = library.Swift.IncludedIDs
	}

	var skippedIDs []string
	if len(module.SkippedIds) > 0 {
		skippedIDs = module.SkippedIds
	} else if library.Swift != nil && len(library.Swift.SkippedIds) > 0 {
		skippedIDs = library.Swift.SkippedIds
	}

	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	return &parser.ModelConfig{
		SpecificationFormat: specFormat,
		SpecificationSource: module.APIPath,
		ServiceConfig:       module.ServiceConfig,
		Source:              sourceConfig,
		Protoc:              pc,
		Override: api.ModelOverride{
			IncludedIDs: includedIDs,
			SkippedIDs:  skippedIDs,
		},
	}
}

func generateSwiftStorage(ctx context.Context, pc *config.Protoc, library *config.Library, module *config.SwiftModule, src *sources.Sources) error {
	storageModule := findModule(library, "google/storage/v2")
	if storageModule == nil {
		return fmt.Errorf("storage module (api_path: google/storage/v2) not found for library %q", library.Name)
	}
	storageModel, err := parser.CreateModel(moduleToModelConfig(library, storageModule, src, pc))
	if err != nil {
		return fmt.Errorf("failed to create storage model: %w", err)
	}

	controlModule := findModule(library, "google/storage/control/v2")
	if controlModule == nil {
		return fmt.Errorf("control module (api_path: google/storage/control/v2) not found for library %q", library.Name)
	}
	controlModel, err := parser.CreateModel(moduleToModelConfig(library, controlModule, src, pc))
	if err != nil {
		return fmt.Errorf("failed to create control model: %w", err)
	}

	return sidekickswift.GenerateStorage(ctx, module.Output, storageModel, storageModule, controlModel, controlModule, library)
}

func findModule(library *config.Library, apiPath string) *config.SwiftModule {
	if library.Swift == nil {
		return nil
	}
	for _, m := range library.Swift.Modules {
		if m.APIPath == apiPath && (m.ModuleType == "grpc-client" || m.ModuleType == "default" || m.ModuleType == "") {
			return m
		}
	}
	return nil
}
