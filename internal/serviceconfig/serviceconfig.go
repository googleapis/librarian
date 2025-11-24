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

// Package serviceconfig reads and parses API service config files.
package serviceconfig

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

// Type aliases for genproto service config types.
type (
	Service            = serviceconfig.Service
	Documentation      = serviceconfig.Documentation
	DocumentationRule  = serviceconfig.DocumentationRule
	Backend            = serviceconfig.Backend
	BackendRule        = serviceconfig.BackendRule
	Authentication     = serviceconfig.Authentication
	AuthenticationRule = serviceconfig.AuthenticationRule
	OAuthRequirements  = serviceconfig.OAuthRequirements
)

// Load loads the service config specified in the configuration.
// If no service config is specified, it returns nil.
func Load(cfg *config.Config) (*Service, error) {
	if name := cfg.General.ServiceConfig; name != "" {
		return Read(FindPath(name, cfg.Source))
	}
	return nil, nil
}

// Read reads a service config from a YAML file and returns it as a Service
// proto. The file is parsed as YAML, converted to JSON, and then unmarshaled
// into a Service proto.
func Read(serviceConfigPath string) (*Service, error) {
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

	cfg := &Service{}
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

// FindPath finds the service config path for the current parser configuration.
//
// The service config files are specified as relative to the `googleapis-root`
// path (or `extra-protos-root` when set). This finds the right path given a
// configuration.
func FindPath(serviceConfigFile string, options map[string]string) string {
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
