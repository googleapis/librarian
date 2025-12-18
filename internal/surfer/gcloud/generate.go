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

	"cloud.google.com/go/librarian/internal/sidekick/config"
	"cloud.google.com/go/librarian/internal/sidekick/parser"
	"cloud.google.com/go/librarian/internal/yaml"
)

// Generate generates gcloud commands for a service.
func Generate(ctx context.Context, googleapis, gcloudconfig, output string) error {
	cfg, err := yaml.Read[Config](gcloudconfig)
	if err != nil {
		return err
	}

	model, err := parser.ParseProtobuf(&config.Config{
		General: config.GeneralConfig{
			// TODO(https://github.com/googleapis/librarian/issues/2817):
			// determine the specification source
			SpecificationSource: "",
		},
		Source: map[string]string{
			"googleapis-root": googleapis,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create API model: %w", err)
	}

	// TODO(https://github.com/googleapis/librarian/issues/2817): implement
	// gcloud command generation logic
	_, _ = model, cfg
	return nil
}
