// Copyright 2026 Google LLC
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

package php

import (
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// Fill populates PHP-specific default values for a library.
func Fill(lib *config.Library) (*config.Library, error) {
	for _, api := range lib.APIs {
		if api.PHP == nil {
			api.PHP = &config.PHPAPI{}
		}
		if api.PHP.CommonResources == nil {
			trueVal := true
			api.PHP.CommonResources = &trueVal
		}
	}
	return lib, nil
}

// Tidy tidies PHP-specific configuration for a library.
func Tidy(lib *config.Library) (*config.Library, error) {
	for _, api := range lib.APIs {
		if api.PHP == nil {
			continue
		}
		if len(api.PHP.AdditionalProtos) > 0 {
			slices.Sort(api.PHP.AdditionalProtos)
			api.PHP.AdditionalProtos = slices.Compact(api.PHP.AdditionalProtos)
		}
		if api.PHP.CommonResources != nil && *api.PHP.CommonResources {
			api.PHP.CommonResources = nil
		}
		empty, err := yaml.Empty(api.PHP)
		if err != nil {
			return nil, err
		}
		if empty {
			api.PHP = nil
		}
	}

	if lib.PHP != nil {
		empty, err := yaml.Empty(lib.PHP)
		if err != nil {
			return nil, err
		}
		if empty {
			lib.PHP = nil
		}
	}

	return lib, nil
}
