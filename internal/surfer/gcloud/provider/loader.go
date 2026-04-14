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

package provider

import (
	"fmt"
	"os"
	"strings"

	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

// CreateAPIModel parses the service specification and creates the API model.
func CreateAPIModel(googleapisPath, includeList, serviceConfig, descriptorFiles, descriptorFilesToGenerate string) (*api.API, error) {
	var includeListSlice []string
	if includeList != "" {
		includeListSlice = strings.Split(includeList, ",")
	}
	parserConfig := &parser.ModelConfig{
		SpecificationFormat:       libconfig.SpecProtobuf,
		ServiceConfig:             serviceConfig,
		DescriptorFiles:           descriptorFiles,
		DescriptorFilesToGenerate: descriptorFilesToGenerate,
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: googleapisPath,
			},
			ActiveRoots: []string{"googleapis"},
			IncludeList: includeListSlice,
		},
	}

	// We use `parser.CreateModel` instead of calling the individual parsing and processing
	// functions directly because CreateModel is the designated entry point that ensures
	// the API model is not only parsed but also fully linked (cross-referenced), validated,
	// and processed with all necessary configuration overrides. This guarantees a complete
	// and consistent model for the generator without code duplication. It's worth noting that
	// we don't use all the functionality of post-processing of CreateModel, so depending
	// on our needs, if we don't find ourselves needing the additional post-processing
	// functionality, we could write our own simpler `CreateModel` function
	model, err := parser.CreateModel(parserConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API model: %w", err)
	}
	return model, nil
}

// ReadGcloudConfig loads the gcloud configuration from a gcloud.yaml file.
func ReadGcloudConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gcloud config file: %w", err)
	}

	cfg, err := yaml.Unmarshal[Config](data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gcloud config YAML: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid gcloud config: %w", err)
	}

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	for i, p := range cfg.ResourcePatterns {
		if p.Type == "" {
			return fmt.Errorf("resource_patterns[%d].type is required", i)
		}
		if len(p.Patterns) == 0 {
			return fmt.Errorf("resource_patterns[%d].patterns must not be empty", i)
		}
	}

	return nil
}
