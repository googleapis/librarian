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

	// Release holds the configuration parameter for any `${lang}-release` subcommand.
	Release *Release `yaml:"release,omitempty"`
}

// Release holds the configuration parameter for publish command.
type Release struct {
	// Remote sets the name of the source-of-truth remote for releases, typically `upstream`.
	Remote string `yaml:"remote,omitempty"`

	// Branch sets the name of the release branch, typically `main`
	Branch string `yaml:"branch,omitempty"`

	// Tools defines the list of tools to install, indexed by installer.
	Tools map[string][]Tool `yaml:"tools,omitempty"`

	// Preinstalled tools defines the list of tools that must be pre-installed.
	//
	// This is indexed by the well-known name of the tool vs. its path, e.g.
	// [preinstalled]
	// cargo = /usr/bin/cargo
	Preinstalled map[string]string `yaml:"preinstalled,omitempty"`

	// IgnoredChanges defines globs that are ignored in change analysis.
	IgnoredChanges []string `yaml:"ignored_changes,omitempty"`

	// An alternative location for the `roots.pem` file. If empty it has no
	// effect.
	RootsPem string `yaml:"roots_pem,omitempty"`
}

// GetExecutablePath finds the path for a given command, checking for an
// override in the configuration first.
func (r *Release) GetExecutablePath(commandName string) string {
	if r != nil && r.Preinstalled != nil {
		if exe, ok := r.Preinstalled[commandName]; ok {
			return exe
		}
	}
	return commandName
}

// Tool defines the configuration required to install helper tools.
type Tool struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version,omitempty"`
}

// Sources references external source repositories.
type Sources struct {
	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`

	// Showcase is the showcase repository configuration.
	Showcase *Source `yaml:"showcase,omitempty"`

	// ProtobufSrc is the path to the `protobuf` repository, used as include directory for `protoc`.
	ProtobufSrc *Source `yaml:"protobuf,omitempty"`

	// Conformance is the path to the `conformance-tests` repository, used as include directory for `protoc`.
	Conformance *Source `yaml:"conformance,omitempty"`
}

// Source represents a source repository.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// SHA256 is the expected hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Branch is the source's git branch to pull updates from.
	// Unset should be interpreted as the repository default branch.
	Branch string `yaml:"branch,omitempty"`

	// Dir is a local directory path to use instead of fetching.
	// If set, Commit and SHA256 are ignored.
	Dir string `yaml:"dir,omitempty"`

	// Subpath is a directory inside the fetched archive that should be treated as
	// the root for operations.
	Subpath string `yaml:"subpath,omitempty"`
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

	// Veneer indicates this library has hand-written code with generated
	// submodules. When true, the library uses language-specific module
	// configuration (e.g., rust.modules) instead of generating a complete crate
	// from channels.
	Veneer bool `yaml:"veneer,omitempty"`

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

	// Roots specifies the source roots to use for generation. Defaults to googleapis.
	Roots []string `yaml:"roots,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc". This
	// overrides Default.Transport.
	Transport string `yaml:"transport,omitempty"`

	// DescriptionOverride overrides the library description.
	DescriptionOverride string `yaml:"description_override,omitempty"`

	// Go contains Go-specific library configuration.
	Go *GoModule `yaml:"go,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`

	// Python contains Python-specific library configuration.
	Python *PythonPackage `yaml:"python,omitempty"`
}

// Channel describes a Channel to include in a library.
type Channel struct {
	// Path specifies which googleapis Path to generate from (for generated
	// libraries).
	Path string `yaml:"path,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`
}
