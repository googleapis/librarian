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
	"github.com/iancoleman/strcase"
)

// Generate generates a Swift client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	if IsMixedLibrary(library) {
		return generateModule(ctx, library, src)
	}
	if len(library.APIs) != 1 {
		return fmt.Errorf("the Swift generator only supports a single api per library")
	}
	modelConfig, err := libraryToModelConfig(library, library.APIs[0], src)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return err
	}
	return sidekickswift.Generate(ctx, model, library.Output, modelConfig, library.Swift)
}

// Format formats a generated Swift library.
func Format(ctx context.Context, library *config.Library) error {
	return command.Run(ctx, "swift-format", "format", "--in-place", "--recursive", library.Output)
}

// DefaultLibraryName derives a library name from an API path.
// For example: google/cloud/secretmanager/v1 -> GoogleCloudSecretmanagerV1.
func DefaultLibraryName(api string) string {
	if clean, ok := strings.CutPrefix(api, "google/cloud/"); ok {
		return "GoogleCloud" + camelLibraryName(clean)
	}
	if clean, ok := strings.CutPrefix(api, "google/"); ok {
		return "Google" + camelLibraryName(clean)
	}
	return "Google" + camelLibraryName(api)
}

func camelLibraryName(api string) string {
	parts := strings.Split(api, "/")
	var name strings.Builder
	for _, p := range parts {
		name.WriteString(strcase.ToCamel(p))
	}
	return name.String()
}

func libraryToModelConfig(library *config.Library, apiCfg *config.API, src *sources.Sources) (*parser.ModelConfig, error) {
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

	modelCfg := &parser.ModelConfig{
		Language:            config.LanguageSwift,
		SpecificationFormat: specFormat,
		ServiceConfig:       svcConfig.ServiceConfig,
		SpecificationSource: apiCfg.Path,
		Source:              sourceConfig,
		Codec: map[string]string{
			"copyright-year": library.CopyrightYear,
			"version":        library.Version,
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
