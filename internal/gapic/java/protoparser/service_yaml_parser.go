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

package protoparser

import (
	"os"

	"go.yaml.in/yaml/v3"
)

// ServiceYamlConfig represents the top-level structure of an API service.yaml file.
type ServiceYamlConfig struct {
	Type          string `yaml:"type"`
	Name          string `yaml:"name"`
	Title         string `yaml:"title"`
	Documentation struct {
		Summary string `yaml:"summary"`
	} `yaml:"documentation"`
	Http struct {
		Rules []struct {
			Selector string `yaml:"selector"`
			Get      string `yaml:"get"`
			Post     string `yaml:"post"`
			Put      string `yaml:"put"`
			Delete   string `yaml:"delete"`
			Patch    string `yaml:"patch"`
			Body     string `yaml:"body"`
		} `yaml:"rules"`
	} `yaml:"http"`
}

// ParseServiceYaml parses a service.yaml file.
func ParseServiceYaml(filePath string) (*ServiceYamlConfig, error) {
	if filePath == "" {
		return nil, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseServiceYamlBytes(content)
}

// ParseServiceYamlBytes parses YAML content for service.yaml.
func ParseServiceYamlBytes(content []byte) (*ServiceYamlConfig, error) {
	var cfg ServiceYamlConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
