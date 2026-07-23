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
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

// ResolveMixinDependencies automatically resolves mixin dependencies for a PHP library.
func ResolveMixinDependencies(cfg *config.Config, lib *config.Library, srcs *sources.Sources) (*config.Config, error) {
	if len(lib.APIs) == 0 {
		return cfg, nil
	}
	for _, apiCfg := range lib.APIs {
		if err := resolveAPIMixinDependencies(lib, apiCfg, srcs); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// resolveAPIMixinDependencies automatically resolves mixin dependencies for a single API.
// It parses the API's service config to identify and add required mixin dependencies
// (e.g., locations, IAM policy) to the API's PHP configuration.
func resolveAPIMixinDependencies(lib *config.Library, apiCfg *config.API, srcs *sources.Sources) error {
	if apiCfg.PHP == nil {
		apiCfg.PHP = &config.PHPAPI{}
	}

	srcCfg := sources.NewSourceConfig(srcs, lib.Roots)
	primaryRoot := srcCfg.Root(srcCfg.ActiveRoots[0])
	mixins, err := serviceconfig.ExtractMixinProtos(primaryRoot, apiCfg.Path, config.LanguagePhp)
	if err != nil {
		return err
	}
	for _, mixinProto := range mixins {
		if !slices.Contains(apiCfg.PHP.AdditionalProtos, mixinProto) {
			apiCfg.PHP.AdditionalProtos = append(apiCfg.PHP.AdditionalProtos, mixinProto)
		}
	}
	sort.Strings(apiCfg.PHP.AdditionalProtos)
	return nil
}
