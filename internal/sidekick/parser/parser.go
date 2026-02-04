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
)

// Config contains the configuration for the parser.
type Config struct {
	SpecificationFormat string
	SpecificationSource string
	ServiceConfig       string
	Source              map[string]string
}

// CreateModel creates a model from the configuration.
func CreateModel(config *Config, overrides *ModelOverrides) (*api.API, error) {
	var model *api.API
	var err error
	switch config.SpecificationFormat {
	case "protobuf":
		model, err = ParseProtobuf(config, overrides)
	case "openapi":
		model, err = ParseOpenAPI(config)
	case "disco", "discovery": // "disco" is legacy
		model, err = ParseDisco(config, overrides)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown specification format: %s", config.SpecificationFormat)
	}

	if err != nil {
		return nil, err
	}
	updateMethodPagination(overrides, model)
	api.LabelRecursiveFields(model)
	if err := api.CrossReference(model); err != nil {
		return nil, err
	}
	if err := api.SkipModelElements(model, overrides.IncludedIDs, overrides.SkippedIDs); err != nil {
		return nil, err
	}
	if err := api.PatchDocumentation(model, overrides.CommentOverrides); err != nil {
		return nil, err
	}
	// Verify all the services, messages and enums are in the same package.
	if err := api.Validate(model); err != nil {
		return nil, err
	}

	if overrides.Name != "" {
		model.Name = overrides.Name
	}
	if overrides.Title != "" {
		model.Title = overrides.Title
	}
	if overrides.Description != "" {
		model.Description = overrides.Description
	}
	return model, nil
}
