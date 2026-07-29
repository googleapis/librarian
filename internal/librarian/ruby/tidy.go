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

package ruby

import (
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// Tidy tidies Ruby-specific configuration for a library.
func Tidy(lib *config.Library) (*config.Library, error) {
	for _, api := range lib.APIs {
		if api.Ruby == nil {
			continue
		}
		if err := tidyAPI(api); err != nil {
			return nil, err
		}
	}
	if err := setNilIfEmpty(&lib.Ruby); err != nil {
		return nil, err
	}
	return lib, nil
}

func tidyAPI(api *config.API) error {
	api.Ruby.AdditionalProtos = tidyAdditionalProtos(api.Ruby.AdditionalProtos)
	if err := setNilIfEmpty(&api.Ruby.RubyCloudOpts); err != nil {
		return err
	}
	if err := setNilIfEmpty(&api.Ruby); err != nil {
		return err
	}
	return nil
}

func tidyAdditionalProtos(protos []string) []string {
	if len(protos) == 0 {
		return nil
	}
	slices.Sort(protos)
	return slices.Compact(protos)
}

func setNilIfEmpty[T any](v *T) error {
	if v == nil {
		return nil
	}
	empty, err := yaml.Empty(*v)
	if err != nil {
		return err
	}
	if empty {
		var zero T
		*v = zero
	}
	return nil
}
