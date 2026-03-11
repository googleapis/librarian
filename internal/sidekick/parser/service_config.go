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
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

func loadServiceConfig(cfg *ModelConfig) (*serviceconfig.Service, error) {
	if cfg.ServiceConfig != "" {
		return serviceconfig.Read(findServiceConfigPath(cfg.ServiceConfig, cfg.Source))
	}
	return nil, nil
}

// findServiceConfigPath finds the service config path for the current parser configuration.
//
// The service config files are specified as relative to the `googleapis`
// path. This finds the right path given a configuration.
func findServiceConfigPath(serviceConfigFile string, sourceCfg config.SourceConfig) string {
	return sourceCfg.Resolve(serviceConfigFile)
}
