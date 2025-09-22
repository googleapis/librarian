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

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/parser/discovery"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ParseDisco reads discovery docs specifications and converts them into
// the `api.API` model.
func ParseDisco(source, serviceConfigFile string, options map[string]string) (*api.API, error) {
	contents, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	var serviceConfig *serviceconfig.Service
	if serviceConfigFile != "" {
		cfg, err := readServiceConfig(findServiceConfigPath(serviceConfigFile, options))
		if err != nil {
			return nil, err
		}
		serviceConfig = cfg
	}
	return discovery.NewAPI(serviceConfig, contents)
}
