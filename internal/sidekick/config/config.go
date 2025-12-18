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

// Package config provides functionality for working with the sidekick.toml
// configuration file.
package config

import (
	"fmt"
	"maps"
	"os"
	"path"

	"cloud.google.com/go/librarian/internal/sidekick/license"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	configName = ".sidekick.toml"
)

// DocumentationOverride describes overrides for the documentation of a single element.
//
// This should be used sparingly. Generally we should prefer updating the
// comments upstream, and then getting a new version of the services
// specification. The exception may be when the fixes take a long time, or are
// specific to one language.
type DocumentationOverride struct {
	ID      string `toml:"id"`
	Match   string `toml:"match"`
	Replace string `toml:"replace"`
}

// PaginationOverride describes overrides for pagination config of a method.
type PaginationOverride struct {
	// The method ID.
	ID string `toml:"id"`
	// The name of the field used for `items`.
	ItemField string `toml:"item-field"`
}

// Config is the main configuration struct.
type Config struct {
	General GeneralConfig `toml:"general"`

	Source              map[string]string       `toml:"source,omitempty"`
	Discovery           *Discovery              `toml:"discovery,omitempty"`
	Codec               map[string]string       `toml:"codec,omitempty"`
	CommentOverrides    []DocumentationOverride `toml:"documentation-overrides,omitempty"`
	PaginationOverrides []PaginationOverride    `toml:"pagination-overrides,omitempty"`
	Release             *Release                `toml:"release,omitempty"`
}

// GeneralConfig contains configuration parameters that affect Parsers and Codecs, including the
// selection of parser and codec.
type GeneralConfig struct {
	Language            string   `toml:"language,omitempty"`
	SpecificationFormat string   `toml:"specification-format,omitempty"`
	SpecificationSource string   `toml:"specification-source,omitempty"`
	ServiceConfig       string   `toml:"service-config,omitempty"`
	IgnoredDirectories  []string `toml:"ignored-directories,omitempty"`
}

// LoadConfig loads the top-level configuration file and validates its contents.
// If no top-level file is found, falls back to the default configuration.
// Where applicable, overrides the top level (or default) configuration values with the ones passed in the command line.
// Returns the merged configuration, or an error if the top level configuration is invalid.
func LoadConfig(language string, source, codec map[string]string) (*Config, error) {
	rootConfig, err := LoadRootConfig(configName)
	if err != nil {
		return nil, err
	}
	argsConfig := &Config{
		General: GeneralConfig{
			Language: language,
		},
		Source: maps.Clone(source),
		Codec:  maps.Clone(codec),
	}
	return mergeConfigs(rootConfig, argsConfig), nil
}

// LoadRootConfig loads the root configuration file.
func LoadRootConfig(filename string) (*Config, error) {
	config := &Config{
		Codec:  map[string]string{},
		Source: map[string]string{},
	}
	if contents, err := os.ReadFile(filename); err == nil {
		err = toml.Unmarshal(contents, &config)
		if err != nil {
			return nil, fmt.Errorf("error reading top-level configuration: %w", err)
		}
	}
	// Ignore errors reading the top-level file.
	return config, nil
}

// MergeConfigAndFile merges the root configuration with a local configuration file.
func MergeConfigAndFile(rootConfig *Config, filename string) (*Config, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var local Config
	err = toml.Unmarshal(contents, &local)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration %s: %w", filename, err)
	}
	return mergeConfigs(rootConfig, &local), nil
}

func mergeConfigs(rootConfig, local *Config) *Config {
	merged := Config{
		General: GeneralConfig{
			Language:            rootConfig.General.Language,
			SpecificationFormat: rootConfig.General.SpecificationFormat,
			IgnoredDirectories:  rootConfig.General.IgnoredDirectories,
		},
		Source:              map[string]string{},
		Codec:               map[string]string{},
		CommentOverrides:    local.CommentOverrides,
		PaginationOverrides: local.PaginationOverrides,
		Discovery:           local.Discovery,
		// Release does not accept local overrides
		Release: rootConfig.Release,
	}
	for k, v := range rootConfig.Codec {
		merged.Codec[k] = v
	}
	for k, v := range rootConfig.Source {
		merged.Source[k] = v
	}

	// Ignore `SpecificationSource` and `ServiceConfig` at the top-level
	// configuration. It makes no sense to set those globally.
	merged.General.SpecificationSource = local.General.SpecificationSource
	merged.General.ServiceConfig = local.General.ServiceConfig
	if local.General.SpecificationFormat != "" {
		merged.General.SpecificationFormat = local.General.SpecificationFormat
	}
	if local.General.Language != "" {
		merged.General.Language = local.General.Language
	}
	for k, v := range local.Codec {
		merged.Codec[k] = v
	}
	for k, v := range local.Source {
		merged.Source[k] = v
	}
	// Ignore errors reading the top-level file.
	return &merged
}

// WriteSidekickToml writes the configuration to a .sidekick.toml file.
func WriteSidekickToml(outDir string, config *Config) error {
	if err := os.MkdirAll(outDir, 0777); err != nil {
		return err
	}
	f, err := os.Create(path.Join(outDir, ".sidekick.toml"))
	if err != nil {
		return err
	}
	defer f.Close()

	year := config.Codec["copyright-year"]
	for _, line := range license.LicenseHeader(year) {
		if line == "" {
			fmt.Fprintln(f, "#")
		} else {
			fmt.Fprintf(f, "#%s\n", line)
		}
	}
	fmt.Fprintln(f, "")

	t := toml.NewEncoder(f)
	if err := t.Encode(config); err != nil {
		return err
	}
	return f.Close()
}
