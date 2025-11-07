// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package automation

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed prod/repositories.yaml
var prodRepositoriesYaml []byte

var availableCommands = map[string]bool{
	"generate":        true,
	"stage-release":   true,
	"publish-release": true,
	"update-image":    true,
}

// RepositoryConfig represents a single registered librarian GitHub repository.
type RepositoryConfig struct {
	Name              string   `yaml:"name"`
	FullName          string   `yaml:"full-name"`
	SecretName        string   `yaml:"github-token-secret-name"`
	SupportedCommands []string `yaml:"supported-commands"`
}

// RepositoriesConfig represents all the registered librarian GitHub repositories.
type RepositoriesConfig struct {
	Repositories []*RepositoryConfig `yaml:"repositories"`
}

// GitURL returns the full git url to clone. If full name is not available,
// it defaults to "https://github.com/googleapis/<name>".
func (c *RepositoryConfig) GitURL() (string, error) {
	if c.FullName == "" {
		if c.Name == "" {
			return "", fmt.Errorf("name or full name is required")
		}
		return fmt.Sprintf("https://github.com/googleapis/%s", c.Name), nil
	}
	return c.FullName, nil
}

// Validate checks the RepositoryConfig is valid.
func (c *RepositoryConfig) Validate() error {
	if c.FullName == "" && c.Name == "" {
		return fmt.Errorf("name or full name is required")
	}
	if c.SecretName == "" {
		return fmt.Errorf("secret name is required")
	}
	if len(c.SupportedCommands) == 0 {
		return fmt.Errorf("supported commands cannot be empty")
	}
	for _, command := range c.SupportedCommands {
		if !availableCommands[command] {
			return fmt.Errorf("unsupported command: %s", command)
		}
	}
	return nil
}

// Validate checks the RepositoriesConfig is valid.
func (c *RepositoriesConfig) Validate() error {
	for i, r := range c.Repositories {
		err := r.Validate()
		if err != nil {
			return fmt.Errorf("invalid repository config at index %d: %w", i, err)
		}
	}
	return nil
}

// RepositoriesForCommand return a subset of repositories that support the provided command.
func (c *RepositoriesConfig) RepositoriesForCommand(command string) []*RepositoryConfig {
	var repositories []*RepositoryConfig
	for _, r := range c.Repositories {
		if slices.Contains(r.SupportedCommands, command) {
			repositories = append(repositories, r)
		}
	}
	return repositories
}

func parseRepositoriesConfig(contentLoader func(file string) ([]byte, error), path string) (*RepositoriesConfig, error) {
	bytes, err := contentLoader(path)
	if err != nil {
		return nil, err
	}
	var c RepositoriesConfig
	if err := yaml.Unmarshal(bytes, &c); err != nil {
		return nil, fmt.Errorf("unmarshaling repositories config state: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("validating repositories config state: %w", err)
	}
	return &c, nil
}

func loadRepositoriesConfig() (*RepositoriesConfig, error) {
	return parseRepositoriesConfig(func(file string) ([]byte, error) { return prodRepositoriesYaml, nil }, "unused")
}
