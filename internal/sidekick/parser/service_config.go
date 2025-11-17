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
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/googleapis/librarian/internal/sidekick/config"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

func loadServiceConfig(cfg *config.Config) (*serviceconfig.Service, error) {
	if name := cfg.General.ServiceConfig; name != "" {
		return readServiceConfig(findServiceConfigPath(name, cfg.Source))
	}
	return nil, nil
}

func readServiceConfig(serviceConfigPath string) (*serviceconfig.Service, error) {
	y, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error reading service config [%s]: %w", serviceConfigPath, err)
	}

	var yamlData interface{}
	if err := yaml.Unmarshal(y, &yamlData); err != nil {
		return nil, fmt.Errorf("error parsing YAML [%s]: %w", serviceConfigPath, err)
	}
	j, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON [%s]: %w", serviceConfigPath, err)
	}

	cfg := &serviceconfig.Service{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(j, cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling service config [%s]: %w", serviceConfigPath, err)
	}

	// An API Service Config will always have a `name` so if it is not populated,
	// it's an invalid config.
	if cfg.GetName() == "" {
		return nil, fmt.Errorf("missing name in service config file [%s]", serviceConfigPath)
	}
	return cfg, nil
}

// findServiceConfigPath finds the service config path for the current parser configuration.
//
// The service config files are specified as relative to the `googleapis-root`
// path (or `extra-protos-root` when set). This finds the right path given a
// configuration.
func findServiceConfigPath(serviceConfigFile string, options map[string]string) string {
	for _, opt := range config.SourceRoots(options) {
		dir, ok := options[opt]
		if !ok {
			// Ignore options that are not set
			continue
		}
		location := path.Join(dir, serviceConfigFile)
		stat, err := os.Stat(location)
		if err == nil && stat.Mode().IsRegular() {
			return location
		}
	}
	// Fallback to the current directory, it may fail but that is detected
	// elsewhere.
	return serviceConfigFile
}
