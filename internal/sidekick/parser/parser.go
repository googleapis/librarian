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

	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
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
	Source              map[string]string

	// Service config
	ServiceConfig string

	// Codec configuration
	Codec map[string]string

	// Documentation/pagination overrides
	CommentOverrides    []api.DocumentationOverride
	PaginationOverrides []api.PaginationOverride

	// Discovery poller configurations
	Discovery *config.Discovery

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
	case libconfig.SpecDiscovery:
		model, err = ParseDisco(cfg)
	case libconfig.SpecOpenAPI:
		model, err = ParseOpenAPI(cfg)
	case libconfig.SpecProtobuf:
		model, err = ParseProtobuf(cfg)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown specification format %q", cfg.SpecificationFormat)
	}
	if err != nil {
		return nil, err
	}
	api.UpdateMethodPagination(cfg.PaginationOverrides, model)
	api.LabelRecursiveFields(model)
	if err := api.CrossReference(model); err != nil {
		return nil, err
	}
	api.IdentifyTargetResources(model)
	if err := api.SkipModelElements(model, cfg.Override); err != nil {
		return nil, err
	}
	if err := api.PatchDocumentation(model, cfg.CommentOverrides); err != nil {
		return nil, err
	}
	// Verify all the services, messages and enums are in the same package.
	if err := api.Validate(model); err != nil {
		return nil, err
	}
	if cfg.Override.Name != "" {
		model.Name = cfg.Override.Name
	}
	if cfg.Override.Title != "" {
		model.Title = cfg.Override.Title
	}
	if cfg.Override.Description != "" {
		model.Description = cfg.Override.Description
	}
	return model, nil
}
