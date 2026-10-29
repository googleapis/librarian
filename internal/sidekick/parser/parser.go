// Copyright 2024 Google LLC
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

package parser

import (
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sources"
)

// ModelConfig holds the configuration necessary to parse an API specification.
type ModelConfig struct {
	Language string

	// Source configuration
	// SpecificationFormat is the format of the API specification.
	// Supported values are:
	// - `config.SpecDiscovery`: "discovery"
	// - `config.SpecOpenAPI`: "openapi"
	// - `config.SpecProtobuf`: "protobuf"
	// - `config.SpecNone`: "none"
	SpecificationFormat string
	SpecificationSource string
	Source              *sources.SourceConfig

	// Protoc configuration
	//
	// This is initialized in the tools configuration for `librarian.yaml`, we want to avoid passing the full
	// librarian configuration to the parsers.
	Protoc *config.Protoc

	// File paths to descriptor files
	DescriptorFilesToGenerate string
	DescriptorFiles           string

	// Service config
	ServiceConfig string

	// Codec configuration
	Codec map[string]string

	// Documentation/pagination overrides
	CommentOverrides    []api.DocumentationOverride
	PaginationOverrides []api.PaginationOverride

	// Discovery poller configurations
	Discovery *api.Discovery

	// Resource heuristic enablement
	ResourceNameHeuristic bool

	// Message module name overrides
	MessageModuleNameOverrides map[string]string

	// Model overrides
	Override api.ModelOverride
}

// CreateModel parses the service specification referenced in `config`,
// cross-references the model, and applies any transformations or overrides
// required by the configuration.
func CreateModel(cfg *ModelConfig) (*api.API, error) {
	var err error
	var model *api.API
	switch cfg.SpecificationFormat {
	case config.SpecDiscovery:
		model, err = ParseDisco(cfg)
	case config.SpecOpenAPI:
		model, err = ParseOpenAPI(cfg)
	case config.SpecProtobuf:
		model, err = ParseProtobuf(cfg)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown specification format %q", cfg.SpecificationFormat)
	}
	if err != nil {
		return nil, err
	}
	return api.NewAPIBuilderFrom(model).
		WithPaginationOverrides(cfg.PaginationOverrides).
		WithResourceNameHeuristic(cfg.ResourceNameHeuristic).
		WithOverride(cfg.Override).
		WithCommentOverrides(cfg.CommentOverrides).
		Build()
}
