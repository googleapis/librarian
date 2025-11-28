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

// Package config provides types and functions for reading and writing
// librarian.yaml configuration files.
package config

import (
	"errors"
)

var errLibraryNotFound = errors.New("library not found")

// Config represents a librarian.yaml configuration file.
type Config struct {
	// Language is the language for this workspace (go, python, rust).
	Language string `yaml:"language"`

	// Repo is the repository name, such as "googleapis/google-cloud-python".
	Repo string `yaml:"repo,omitempty"`

	// Sources references external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Default contains default settings for all libraries.
	Default *Default `yaml:"default,omitempty"`

	// Libraries contains configuration overrides for libraries that need
	// special handling, and differ from default settings.
	Libraries []*Library `yaml:"libraries,omitempty"`
}

// Sources references external source repositories.
type Sources struct {
	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents a source repository.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// SHA256 is the expected hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Dir is a local directory path to use instead of fetching. If set, Commit
	// and SHA256 are ignored.
	Dir string `yaml:"dir,omitempty"`
}

// Default contains default settings for all libraries.
type Default struct {
	// Output is the directory where code is written. For example, for Rust
	// this is src/generated.
	Output string `yaml:"output,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc".
	Transport string `yaml:"transport,omitempty"`

	// ReleaseLevel is either "stable" or "preview".
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// TagFormat is the template for git tags, such as "{name}/v{version}".
	TagFormat string `yaml:"tag_format,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// Library represents a library configuration.
type Library struct {
	// Name is the library name, such as "secretmanager" or "storage". It is
	// listed first so it appears at the top of each library entry in YAML.
	Name string `yaml:"name"`

	// Channel specifies which googleapis Channel to generate from (for generated
	// libraries).
	Channels []*Channel `yaml:"channels,omitempty"`

	// SkipGenerate disables code generation for this library.
	SkipGenerate bool `yaml:"skip_generate,omitempty"`

	// SkipRelease disables releasing for this library.
	SkipRelease bool `yaml:"skip_release,omitempty"`

	// SkipPublish disables publishing for this library.
	SkipPublish bool `yaml:"skip_publish,omitempty"`

	// Output is the directory where code is written. This overrides
	// Default.Output.
	Output string `yaml:"output,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// Keep lists files and directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// ReleaseLevel is the release level, such as "stable" or "preview". This
	// overrides Default.ReleaseLevel.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// SpecificationFormat specifies the API specification format. Valid values
	// are "protobuf" (default) or "discovery".
	SpecificationFormat string `yaml:"specification_format,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc". This
	// overrides Default.Transport.
	Transport string `yaml:"transport,omitempty"`

	// DescriptionOverride overrides the library description.
	DescriptionOverride string `yaml:"description_override,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`
}

// Channel describes a Channel to include in a library.
type Channel struct {
	// Path specifies which googleapis Path to generate from (for generated
	// libraries).
	Path string `yaml:"path,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`
}

// LibraryByName returns a library with the given name.
func (c *Config) LibraryByName(name string) (*Library, error) {
	if c.Libraries == nil {
		return nil, errLibraryNotFound
	}

	for _, library := range c.Libraries {
		if library.Name == name {
			return library, nil
		}
	}

	return nil, errLibraryNotFound
}

// Fill populates empty library fields from the provided defaults.
func (lib *Library) Fill(d *Default) {
	if d == nil {
		return
	}
	if lib.Output == "" {
		lib.Output = d.Output
	}
	if lib.ReleaseLevel == "" {
		lib.ReleaseLevel = d.ReleaseLevel
	}
	if lib.Transport == "" {
		lib.Transport = d.Transport
	}
	if d.Rust != nil {
		lib.fillRust(d)
	}
}

// fillRust() populates empty fields in `lib.Rust` from the provided default.
func (lib *Library) fillRust(d *Default) {
	if lib.Rust == nil {
		lib.Rust = &RustCrate{}
	}
	lib.Rust.PackageDependencies = mergePackageDependencies(
		d.Rust.PackageDependencies,
		lib.Rust.PackageDependencies,
	)
	if len(lib.Rust.DisabledRustdocWarnings) == 0 {
		lib.Rust.DisabledRustdocWarnings = d.Rust.DisabledRustdocWarnings
	}
}

// mergePackageDependencies merges default and library package dependencies,
// with library dependencies taking precedence for duplicates.
func mergePackageDependencies(defaults, lib []*RustPackageDependency) []*RustPackageDependency {
	seen := make(map[string]bool)
	var result []*RustPackageDependency
	for _, dep := range lib {
		seen[dep.Name] = true
		result = append(result, dep)
	}
	for _, dep := range defaults {
		if seen[dep.Name] {
			continue
		}
		copied := *dep
		result = append(result, &copied)
	}
	return result
}
