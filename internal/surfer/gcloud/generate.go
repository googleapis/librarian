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

package gcloud

import (
	"context"
	"fmt"
	"os"

	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"gopkg.in/yaml.v3"
)

// Generate generates gcloud commands for a service.
func Generate(ctx context.Context, googleapis, gcloudconfig, output string) error {
	cfg, err := readGcloudConfig(gcloudconfig)
	if err != nil {
		return fmt.Errorf("failed to load gcloud config: %w", err)
	}

	// TODO(https://github.com/googleapis/librarian/issues/2817): determine the
	// specification source path
	var source string

	sidekickcfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			SpecificationSource: source,
		},
		Source: map[string]string{
			"googleapis-root": googleapis,
		},
	}
	model, err := parser.ParseProtobuf(sidekickcfg)
	if err != nil {
		return fmt.Errorf("failed to create API model: %w", err)
	}

	// TODO(https://github.com/googleapis/librarian/issues/2817): implement
	// gcloud command generation logic.
	_, _ = model, cfg

	return nil
}

// readGcloudConfig loads the gcloud configuration from a gcloud.yaml file.
func readGcloudConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gcloud config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse gcloud config YAML: %w", err)
	}
	return &cfg, nil
}
