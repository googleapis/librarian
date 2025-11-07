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

// CreateModel parses the service specification referenced in `config`,
// cross-references the model, and applies any transformations or overrides
// required by the configuration.
func CreateModel(config *config.Config) (*api.API, error) {
	var err error
	var model *api.API
	switch config.General.SpecificationFormat {
	case "disco":
		model, err = ParseDisco(config)
	case "openapi":
		model, err = ParseOpenAPI(config)
	case "protobuf":
		model, err = ParseProtobuf(config)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown parser %q", config.General.SpecificationFormat)
	}
	if err != nil {
		return nil, err
	}
	updateMethodPagination(config.PaginationOverrides, model)
	api.LabelRecursiveFields(model)
	if err := api.CrossReference(model); err != nil {
		return nil, err
	}
	if err := api.SkipModelElements(model, config.Source); err != nil {
		return nil, err
	}
	if err := api.PatchDocumentation(model, config); err != nil {
		return nil, err
	}
	// Verify all the services, messages and enums are in the same package.
	if err := api.Validate(model); err != nil {
		return nil, err
	}
	if name, ok := config.Source["name-override"]; ok {
		model.Name = name
	}
	if title, ok := config.Source["title-override"]; ok {
		model.Title = title
	}
	if description, ok := config.Source["description-override"]; ok {
		model.Description = description
	}
	return model, nil
}
