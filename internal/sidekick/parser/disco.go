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

package parser

import (
	"os"
	"path"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser/discovery"
)

// ParseDisco reads discovery docs specifications and converts them into
// the `api.API` model.
func ParseDisco(cfg ModelConfig) (*api.API, error) {
	source := cfg.SpecificationSource
	for _, opt := range config.SourceRoots(cfg.Source) {
		location, ok := cfg.Source[opt]
		if !ok {
			// Ignore options that are not set
			continue
		}
		fullName := path.Join(location, source)
		if _, err := os.Stat(fullName); err == nil {
			source = fullName
			break
		}
	}
	contents, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	serviceConfig, err := loadServiceConfig(cfg)
	if err != nil {
		return nil, err
	}
	result, err := discovery.NewAPI(serviceConfig, contents, cfg.Discovery)
	if err != nil {
		return nil, err
	}
	updateAutoPopulatedFields(serviceConfig, result)
	return result, nil
}
