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

//go:generate go run -tags configdocgen ../../cmd/config_doc_generate.go -input . -output ../../doc/config-schema.md

// Config represents a librarian.yaml configuration file.
type Config struct {
	// Language is the language for this workspace (go, python, rust).
	Language string `yaml:"language"`

	// Version is the librarian tool version to use.
	Version string `yaml:"version,omitempty"`

	// Repo is the repository name, such as "googleapis/google-cloud-python".
	//
	// TODO(https://github.com/googleapis/librarian/issues/3003): Remove this
	// field when .repo-metadata.json generation is removed.
	Repo string `yaml:"repo,omitempty"`

	// Sources references external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Release holds the configuration parameter for publishing and release subcommands.
	Release *Release `yaml:"release,omitempty"`

	// Default contains default settings for all libraries. They apply to all libraries unless overridden.
	Default *Default `yaml:"default,omitempty"`

	// Libraries contains configuration overrides for libraries that need
	// special handling, and differ from default settings.
	Libraries []*Library `yaml:"libraries,omitempty"`
}

// Release holds the configuration parameter for publish command.
type Release struct {
	// Branch sets the name of the release branch, typically `main`
	Branch string `yaml:"branch,omitempty"`

	// IgnoredChanges defines globs that are ignored in change analysis.
	IgnoredChanges []string `yaml:"ignored_changes,omitempty"`

	// Preinstalled tools defines the list of tools that must be preinstalled.
	//
	// This is indexed by the well-known name of the tool vs. its path, e.g.
	// [preinstalled]
	// cargo = /usr/bin/cargo
	Preinstalled map[string]string `yaml:"preinstalled,omitempty"`

	// Remote sets the name of the source-of-truth remote for releases, typically `upstream`.
	Remote string `yaml:"remote,omitempty"`

	// An alternative location for the `roots.pem` file. If empty it has no
	// effect.
	RootsPem string `yaml:"roots_pem,omitempty"`

	// Tools defines the list of tools to install, indexed by installer.
	Tools map[string][]Tool `yaml:"tools,omitempty"`
}

// Tool defines the configuration required to install helper tools.
type Tool struct {
	// Name is the name of the tool e.g. nox.
	Name string `yaml:"name"`

	// Version is the version of the tool e.g. 1.2.4.
	Version string `yaml:"version,omitempty"`
}

// Sources references external source repositories.
type Sources struct {
	// Conformance is the path to the `conformance-tests` repository, used as include directory for `protoc`.
	Conformance *Source `yaml:"conformance,omitempty"`

	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`

	// ProtobufSrc is the path to the `protobuf` repository, used as include directory for `protoc`.
	ProtobufSrc *Source `yaml:"protobuf,omitempty"`

	// Showcase is the showcase repository configuration.
	Showcase *Source `yaml:"showcase,omitempty"`
}

// Source represents a source repository.
type Source struct {
	// Branch is the source's git branch to pull updates from.
	// Unset should be interpreted as the repository default branch.
	Branch string `yaml:"branch,omitempty"`

	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// Dir is a local directory path to use instead of fetching.
	// If set, Commit and SHA256 are ignored.
	Dir string `yaml:"dir,omitempty"`

	// SHA256 is the expected hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Subpath is a directory inside the fetched archive that should be treated as
	// the root for operations.
	Subpath string `yaml:"subpath,omitempty"`
}

// Default contains default settings for all libraries.
type Default struct {
	// Keep lists files and directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`
	// Output is the directory where code is written. For example, for Rust
	// this is src/generated.
	Output string `yaml:"output,omitempty"`

	// ReleaseLevel is either "stable" or "preview".
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// TagFormat is the template for git tags, such as "{name}/v{version}".
	TagFormat string `yaml:"tag_format,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc".
	Transport string `yaml:"transport,omitempty"`

	// Language-specific fields are below.

	// Dart contains Dart-specific default configuration.
	Dart *DartPackage `yaml:"dart,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// Library represents a library configuration.
type Library struct {
	// Note: Properties should typically be added in alphabetical order, but
	// because this order impacts YAML serialization, we keep Name and Version
	// at the top for ease of consumption in file-form.

	// Name is the library name, such as "secretmanager" or "storage".
	Name string `yaml:"name"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// API specifies which googleapis API to generate from (for generated
	// libraries).
	APIs []*API `yaml:"apis,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// DescriptionOverride overrides the library description.
	DescriptionOverride string `yaml:"description_override,omitempty"`

	// Keep lists files and directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// Output is the directory where code is written. This overrides
	// Default.Output.
	Output string `yaml:"output,omitempty"`

	// ReleaseLevel is the release level, such as "stable" or "preview". This
	// overrides Default.ReleaseLevel.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Roots specifies the source roots to use for generation. Defaults to googleapis.
	Roots []string `yaml:"roots,omitempty"`

	// SkipGenerate disables code generation for this library.
	SkipGenerate bool `yaml:"skip_generate,omitempty"`

	// SkipRelease disables release for this library.
	SkipRelease bool `yaml:"skip_release,omitempty"`

	// SpecificationFormat specifies the API specification format. Valid values
	// are "protobuf" (default) or "discovery".
	SpecificationFormat string `yaml:"specification_format,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc". This
	// overrides Default.Transport.
	Transport string `yaml:"transport,omitempty"`

	// Veneer indicates this library has handwritten code. A veneer may
	// contain generated libraries.
	Veneer bool `yaml:"veneer,omitempty"`

	// Language-specific fields are below.

	// Dart contains Dart-specific library configuration.
	Dart *DartPackage `yaml:"dart,omitempty"`

	// Go contains Go-specific library configuration.
	Go *GoModule `yaml:"go,omitempty"`

	// Python contains Python-specific library configuration.
	Python *PythonPackage `yaml:"python,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`
}

// API describes an API to include in a library.
type API struct {
	// Path specifies which googleapis Path to generate from (for generated
	// libraries).
	Path string `yaml:"path,omitempty"`
}
