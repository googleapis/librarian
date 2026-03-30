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

// Package gcloud provides functionality for generating gcloud commands.
package gcloud

import (
	"context"

	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

// GenerateConfig contains parameters for generating gcloud commands.
type GenerateConfig struct {
	Googleapis    string
	GcloudConfig  string
	Output        string
	IncludeList   string
	ServiceConfig string
}

// Generate generates gcloud commands for a service.
func Generate(_ context.Context, cfg GenerateConfig) error {
	overrides, err := provider.ReadGcloudConfig(cfg.GcloudConfig)
	if err != nil {
		return err
	}

	model, err := provider.CreateAPIModel(cfg.Googleapis, cfg.IncludeList, cfg.ServiceConfig)
	if err != nil {
		return err
	}

	rootGroup, err := newCommandTreeBuilder(model, overrides).build()
	if err != nil {
		return err
	}

	return writeCommandGroupTree(cfg.Output, rootGroup)
}
