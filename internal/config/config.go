// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete librarian.yaml configuration file.
type Config struct {
	// Language is the primary language for this repository (go, python, rust).
	Language string `yaml:"language"`

	// Repo is the repository name (e.g., "googleapis/google-cloud-python").
	Repo string `yaml:"repo,omitempty"`

	// Sources contains references to external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Default contains default generation settings.
	Default *Default `yaml:"default"`

	// Libraries contains configuration overrides for libraries that need special handling.
	// Only include libraries that differ from defaults.
	// Versions are looked up from the Versions map below.
	Libraries []*Library `yaml:"libraries,omitempty"`

	// Versions contains version numbers for all libraries.
	// This is the source of truth for release versions.
	// Key is library name, value is version string.
	Versions map[string]string `yaml:"versions,omitempty"`
}

// Sources contains references to external source repositories.
// Each entry maps a source name to its configuration.
type Sources struct {
	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents a single source repository configuration.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// SHA256 is the expected SHA256 hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Dir is a local directory path to use instead of fetching.
	// This is useful for testing. If set, Commit and SHA256 are ignored.
	Dir string `yaml:"-"`
}

// Default contains default generation settings.
type Default struct {
	// Output is the directory where generated code is written (relative to repository root).
	Output string `yaml:"output,omitempty"`

	// Generate contains default generation configuration.
	Generate *DefaultGenerate `yaml:"generate,omitempty"`

	// Release contains default release configuration.
	Release *DefaultRelease `yaml:"release,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// DefaultGenerate contains default generation configuration.
type DefaultGenerate struct {

	// Transport is the default transport protocol (e.g., "grpc+rest", "grpc").
	Transport string `yaml:"transport,omitempty"`

	// ReleaseLevel is the default release level ("stable" or "preview").
	ReleaseLevel string `yaml:"release_level,omitempty"`
}

// DefaultRelease contains release configuration.
type DefaultRelease struct {
	// TagFormat is the template for git tags (e.g., '{name}/v{version}').
	// Supported placeholders: {name}, {version}
	TagFormat string `yaml:"tag_format,omitempty"`

	// Remote is the git remote name (e.g., "upstream", "origin").
	Remote string `yaml:"remote,omitempty"`

	// Branch is the default branch for releases (e.g., "main", "master").
	Branch string `yaml:"branch,omitempty"`
}

// Library represents a single library configuration entry.
type Library struct {
	// Name is the library name (e.g., "secretmanager", "storage").
	Name string `yaml:"name,omitempty"`

	// Channel specifies which googleapis Channel to generate from (for generated libraries).
	// Can be a string (protobuf Channel path) or an APIObject (for discovery APIs).
	// If both Channel and APIs are empty, this is a handwritten library.
	Channel string `yaml:"channel,omitempty"`

	// Channels specifies multiple API versions to bundle into one library (for multi-version libraries).
	// Alternative to API field for libraries that bundle multiple versions.
	Channels []string `yaml:"channels,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// Generate contains per-library generate configuration.
	Generate *LibraryGenerate `yaml:"generate,omitempty"`

	// Keep lists files/directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// Output specifies the filesystem location (overrides computed location from defaults.output).
	// For generated libraries: overrides where code is generated to.
	// For handwritten libraries: specifies the source directory.
	Output string `yaml:"output,omitempty"`

	// Publish contains per-library publish configuration.
	Publish *LibraryPublish `yaml:"publish,omitempty"`

	// Release contains per-library release configuration.
	Release *LibraryRelease `yaml:"release,omitempty"`

	// ReleaseLevel overrides the default release level.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`

	// SpecificationFormat specifies the API specification format.
	// Valid values are "protobuf" (default) or "discovery".
	SpecificationFormat string `yaml:"specification_format,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"-"`

	// Transport overrides the default transport.
	Transport string `yaml:"transport,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`
}

// LibraryGenerate contains per-library generate configuration.
type LibraryGenerate struct {
	// Disabled prevents library generation.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryRelease contains per-library release configuration.
type LibraryRelease struct {
	// Disabled prevents library release.
	Disabled bool `yaml:"disabled,omitempty"`
}

// LibraryPublish contains per-library publish configuration.
type LibraryPublish struct {
	// Disabled prevents library from being published to package registries.
	Disabled bool `yaml:"disabled,omitempty"`
}

// Read reads the configuration from a file.
func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &c, nil
}

// Write writes the configuration to a file.
func (c *Config) Write(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	defer enc.Close()

	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return nil
}
