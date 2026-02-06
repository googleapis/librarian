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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

// ModelConfig holds the configuration necessary to parse an API specification.
type ModelConfig struct {
	Language string

	// Source configuration
	// SpecificationFormat is the format of the API specification.
	// Supported values are "discovery", "openapi", "protobuf", and "none".
	SpecificationFormat string
	SpecificationSource string
	Source              map[string]string

	// Service config
	ServiceConfig string

	// Codec configuration
	Codec map[string]string

	// Documentation/pagination overrides
	CommentOverrides    []config.DocumentationOverride
	PaginationOverrides []config.PaginationOverride

	// Discovery poller configurations
	Discovery *config.Discovery
}

// NewModelConfigFromSidekickConfig creates a ModelConfig from a sidekick Config.
func NewModelConfigFromSidekickConfig(cfg *config.Config) ModelConfig {
	if cfg == nil {
		return ModelConfig{}
	}
	specFormat := cfg.General.SpecificationFormat
	if specFormat == "disco" {
		specFormat = "discovery"
	}
	return ModelConfig{
		Language:            cfg.General.Language,
		SpecificationFormat: specFormat,
		SpecificationSource: cfg.General.SpecificationSource,
		ServiceConfig:       cfg.General.ServiceConfig,
		Source:              cfg.Source,
		Codec:               cfg.Codec,
		CommentOverrides:    cfg.CommentOverrides,
		PaginationOverrides: cfg.PaginationOverrides,
		Discovery:           cfg.Discovery,
	}
}

// CreateModel parses the service specification referenced in `config`,
// cross-references the model, and applies any transformations or overrides
// required by the configuration.
func CreateModel(cfg ModelConfig) (*api.API, error) {
	var err error
	var model *api.API
	switch cfg.SpecificationFormat {
	case "discovery":
		model, err = ParseDisco(cfg)
	case "openapi":
		model, err = ParseOpenAPI(cfg)
	case "protobuf":
		model, err = ParseProtobuf(cfg)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown specification format %q", cfg.SpecificationFormat)
	}
	if err != nil {
		return nil, err
	}
	updateMethodPagination(cfg.PaginationOverrides, model)
	api.LabelRecursiveFields(model)
	if err := api.CrossReference(model); err != nil {
		return nil, err
	}
	if err := api.SkipModelElements(model, cfg.Source); err != nil {
		return nil, err
	}
	if err := api.PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		return nil, err
	}
	// Verify all the services, messages and enums are in the same package.
	if err := api.Validate(model); err != nil {
		return nil, err
	}
	if name, ok := cfg.Source["name-override"]; ok {
		model.Name = name
	}
	if title, ok := cfg.Source["title-override"]; ok {
		model.Title = title
	}
	if description, ok := cfg.Source["description-override"]; ok {
		model.Description = description
	}
	return model, nil
}
