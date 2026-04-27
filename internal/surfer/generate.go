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

package surfer

import (
	"github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
)

// generateConfig contains parameters for generating gcloud commands.
type generateConfig struct {
	GcloudConfig              string
	ServiceConfig             string
	IncludeList               string
	Googleapis                string
	DescriptorFilesToGenerate string
	DescriptorFiles           string
	Output                    string
	BaseModule                string
}

// generate generates gcloud commands for a service.
func generate(cfg generateConfig) error {
	overrides, err := provider.ReadGcloudConfig(cfg.GcloudConfig)
	if err != nil {
		return err
	}
	model, err := provider.CreateAPIModel(cfg.Googleapis, cfg.IncludeList, cfg.ServiceConfig, cfg.DescriptorFiles, cfg.DescriptorFilesToGenerate)
	if err != nil {
		return err
	}
	return gcloud.Generate(model, overrides, cfg.Output, cfg.BaseModule)
}
