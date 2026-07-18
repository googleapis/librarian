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

const (
	// BranchMain is the default git branch name.
	BranchMain = "main"

	// LibrarianYAML is the filename for the librarian configuration file.
	LibrarianYAML = "librarian.yaml"

	// RemoteUpstream is the default git remote name.
	RemoteUpstream = "upstream"
)

// Config represents a librarian.yaml configuration file.
type Config struct {
	// Language is the language for this workspace (go, python, rust).
	Language string `yaml:"language"`

	// Version is the librarian tool version to use.
	Version string `yaml:"version,omitempty"`

	// Repo is the repository name, such as "googleapis/google-cloud-python".
	// It is used for:
	// - Providing to the Java GAPIC generator for observability features.
	// - Generating the .repo-metadata.json.
	Repo string `yaml:"repo,omitempty"`

	// Sources references external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Tools defines required tools.
	Tools *Tools `yaml:"tools,omitempty"`

	// Default contains default settings for all libraries. They apply to all libraries unless overridden.
	Default *Default `yaml:"default,omitempty"`

	// Libraries contains configuration overrides for libraries that need
	// special handling, and differ from default settings.
	Libraries []*Library `yaml:"libraries,omitempty"`
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

// Tools defines required tools.
type Tools struct {
	// Cargo defines tools to install via cargo.
	Cargo []*CargoTool `yaml:"cargo,omitempty"`

	// Composer defines tools to install via Composer.
	Composer []*ComposerTool `yaml:"composer,omitempty"`

	// Go defines tools to install via go.
	Go []*GoTool `yaml:"go,omitempty"`

	// Gem defines tools to install via gem.
	Gem []*GemTool `yaml:"gem,omitempty"`

	// Maven defines tools to install via Maven.
	Maven []*MavenTool `yaml:"maven,omitempty"`

	// Pip defines tools to install via pip.
	Pip []*PipTool `yaml:"pip,omitempty"`

	// PNPM defines tools to install via pnpm.
	PNPM []*PNPMTool `yaml:"pnpm,omitempty"`

	// Protoc defines the protoc installation.
	Protoc *Protoc `yaml:"protoc,omitempty"`
}

// CargoTool defines a tool to install via cargo.
type CargoTool struct {
	// Name is the cargo package name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version"`
}

// ComposerTool defines a tool to install via Composer.
type ComposerTool struct {
	// Name is the composer package name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version"`

	// Repo is the GitHub repository to fetch the tool from (e.g. github.com/googleapis/gapic-generator-php).
	Repo string `yaml:"repo,omitempty"`

	// SHA256 is the SHA256 checksum of the package.
	SHA256 string `yaml:"sha256,omitempty"`
}

// GemTool defines a tool to install via gem.
type GemTool struct {
	// Name is the gem name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version"`
}

// GoTool defines a tool to install via go.
type GoTool struct {
	// Name is the go module name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version,omitempty"`
}

// MavenTool defines a tool to install via Maven.
type MavenTool struct {
	// Name is the Maven tool name. It is used as the filename for the generated executable wrapper script.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version,omitempty"`

	// GroupID is the Maven artifact group ID.
	GroupID string `yaml:"group_id,omitempty"`

	// ArtifactID is the Maven artifact ID.
	ArtifactID string `yaml:"artifact_id,omitempty"`

	// Classifier is the classifier of the Maven artifact.
	Classifier string `yaml:"classifier,omitempty"`

	// Packaging is the Maven packaging. Acceptable values are lowercase "jar" and "exe".
	// If the packaging is "exe", the wrapper script executes it directly.
	// Otherwise, it executes the tool using "java -jar".
	Packaging string `yaml:"packaging,omitempty"`

	// LocalPath is the path to a local Maven project directory containing a pom.xml file.
	// When present, version, group_id, artifact_id are ignored.
	LocalPath string `yaml:"local_path,omitempty"`

	// MainClass is the fully qualified main class name to execute (used with -cp).
	MainClass string `yaml:"main_class,omitempty"`
}

// PipTool defines a tool to install via pip.
type PipTool struct {
	// Name is the pip package name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version"`

	// Package is the pip install specifier (e.g., "pkg@git+https://...").
	Package string `yaml:"package,omitempty"`

	// LocalPath is the path to a local Python package to install.
	LocalPath string `yaml:"local_path,omitempty"`
}

// PNPMTool defines a tool to install via pnpm.
type PNPMTool struct {
	// Name is the pnpm package name.
	Name string `yaml:"name"`

	// Version is the version to install.
	Version string `yaml:"version"`

	// Package is the URL or path of the package to install.
	Package string `yaml:"package,omitempty"`

	// SHA256 is the SHA256 checksum of the package.
	SHA256 string `yaml:"sha256,omitempty"`

	// Checksum is a deprecated alias for SHA256.
	Checksum string `yaml:"checksum,omitempty"`

	// Build defines the commands to run to build the tool after installation.
	Build []string `yaml:"build,omitempty"`

	// SrcDir is the path to the directory inside the fetched archive that should
	// be treated as the root for operations.
	SrcDir string `yaml:"src_dir,omitempty"`
}

// Protoc defines the configuration for installing the protoc compiler.
type Protoc struct {
	// Version is the version to install.
	Version string `yaml:"version,omitempty"`

	// SHA256 is the SHA256 checksum of the tarball.
	SHA256 string `yaml:"sha256,omitempty"`
}

// Default contains default settings for all libraries.
type Default struct {
	// Keep lists files and directories to preserve during regeneration. These represent
	// critical custom handwritten files (e.g., package.json, custom configs, and handwritten tests)
	// and semi-handmade documentation files (README.md, CHANGELOG.md, .readme-partials.yaml)
	// that are not natively generated from proto schemas but are strictly required by the post-processor's
	// markdown generation and release tracking passes.
	Keep []string `yaml:"keep,omitempty"`
	// Output is the directory where code is written. For example, for Rust
	// this is src/generated.
	Output string `yaml:"output,omitempty"`

	// TagFormat is the template for git tags, such as "{name}/v{version}".
	TagFormat string `yaml:"tag_format,omitempty"`

	// Language-specific fields are below.

	// Dart contains Dart-specific default configuration.
	Dart *DartPackage `yaml:"dart,omitempty"`

	// Dotnet contains .NET-specific default configuration.
	Dotnet *DotnetPackage `yaml:"dotnet,omitempty"`

	// Go contains Go-specific default configuration.
	Go *GoDefault `yaml:"go,omitempty"`

	// Java contains Java-specific default configuration.
	Java *JavaDefault `yaml:"java,omitempty"`

	// Nodejs contains Node.js-specific default configuration.
	Nodejs *NodejsPackage `yaml:"nodejs,omitempty"`

	// PHP contains PHP-specific default configuration.
	PHP *PHPDefault `yaml:"php,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`

	// Python contains Python-specific default configuration.
	Python *PythonDefault `yaml:"python,omitempty"`

	// Swift contains Swift-specific default configuration.
	Swift *SwiftDefault `yaml:"swift,omitempty"`
}

// Library represents a library configuration.
type Library struct {
	// Note: Properties should typically be added in alphabetical order, but
	// because this order impacts YAML serialization, we keep Name and Version
	// at the top for ease of consumption in file-form.

	// Name is the library name, such as "secretmanager" or "storage".
	Name string `yaml:"name,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// Preview signifies that this API has a preview variant, and it contains
	// overrides specific to the preview API variant. This is merged with the
	// containing [Library], preferring those [Library.Preview] values that are
	// set over their counterpart in the containing configuration.
	//
	// The most common overrides are [Library.Version] and [Library.APIs], with
	// the former containing a pre-release version based on the containing
	// version of the stable client, and the latter being a subset of APIs,
	// typically omitting alpha and beta paths.
	//
	// The [Library.Output] may be a different location and derived on a
	// per-language basis, but will not be serialized in the configuration.
	//
	// Important: The boolean fields [Library.SkipRelease] and
	// [Library.SkipGenerate] set in the containing config will always be
	// applied to the Preview library as well, because previews are related to
	// the stable library and should be managed identically.
	Preview *Library `yaml:"preview,omitempty"`

	// API specifies which googleapis API to generate from (for generated
	// libraries).
	APIs []*API `yaml:"apis,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// TitleOverride overrides the title used in README generation.
	TitleOverride string `yaml:"title_override,omitempty"`

	// Keep lists files and directories to preserve during regeneration. These represent
	// critical custom handwritten files (e.g., package.json, custom configs, and handwritten tests)
	// and semi-handmade documentation files (README.md, CHANGELOG.md, .readme-partials.yaml)
	// that are not natively generated from proto schemas but are strictly required by the post-processor's
	// markdown generation and release tracking passes.
	Keep []string `yaml:"keep,omitempty"`

	// Output is the directory where code is written. This overrides
	// Default.Output.
	Output string `yaml:"output,omitempty"`

	// Postprocess contains post-processing operations executed after code generation.
	Postprocess *Postprocess `yaml:"postprocess,omitempty"`

	// Roots specifies the source roots to use for generation. Defaults to googleapis.
	Roots []string `yaml:"roots,omitempty"`

	// SkipGenerate disables code generation for this library.
	SkipGenerate bool `yaml:"skip_generate,omitempty"`

	// SkipRelease disables release for this library.
	SkipRelease bool `yaml:"skip_release,omitempty"`

	// SpecificationFormat specifies the API specification format. Valid values
	// are "protobuf" (default) or "discovery".
	SpecificationFormat string `yaml:"specification_format,omitempty"`

	// Language-specific fields are below.

	// Dart contains Dart-specific library configuration.
	Dart *DartPackage `yaml:"dart,omitempty"`

	// Dotnet contains .NET-specific library configuration.
	Dotnet *DotnetPackage `yaml:"dotnet,omitempty"`

	// Go contains Go-specific library configuration.
	Go *GoModule `yaml:"go,omitempty"`

	// Java contains Java-specific library configuration.
	Java *JavaModule `yaml:"java,omitempty"`

	// Nodejs contains Node.js-specific library configuration.
	Nodejs *NodejsPackage `yaml:"nodejs,omitempty"`

	// PHP contains PHP-specific library configuration.
	PHP *PHPPackage `yaml:"php,omitempty"`

	// Python contains Python-specific library configuration.
	Python *PythonPackage `yaml:"python,omitempty"`

	// Ruby contains Ruby-specific library configuration.
	Ruby *RubyPackage `yaml:"ruby,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`

	// Swift contains Swift-specific library configuration.
	Swift *SwiftPackage `yaml:"swift,omitempty"`
}

// Postprocess represents post-processing configuration options integrated into librarian.yaml.
type Postprocess struct {
	// Replace contains literal string replacement rules.
	Replace []ReplaceConfig `yaml:"replace,omitempty"`

	// ReplaceRegex contains regular expression replacement rules.
	ReplaceRegex []ReplaceRegexConfig `yaml:"replace_regex,omitempty"`

	// CopyFile contains file copy rules.
	CopyFile []CopyConfig `yaml:"copy_file,omitempty"`

	// RemoveFile contains glob patterns of files to remove.
	RemoveFile []string `yaml:"remove_file,omitempty"`

	// MethodOperations contains method-level operations (`delete`, `duplicate`, `deprecate`).
	MethodOperations []MethodOperation `yaml:"method_operations,omitempty"`
}

// MethodOperation represents a method-level operation like delete, duplicate, or deprecate.
type MethodOperation struct {
	// Path specifies the relative file path to modify.
	Path string `yaml:"path"`

	// Action specifies the operation (`delete`, `duplicate`, or `deprecate`).
	Action string `yaml:"action"`

	// FuncName specifies the target method name.
	FuncName string `yaml:"func_name"`

	// NewName specifies the new method name for duplicate operations.
	NewName string `yaml:"new_name,omitempty"`

	// DeprecationMessage specifies the deprecation message for deprecate operations.
	DeprecationMessage string `yaml:"deprecation_message,omitempty"`
}

// ReplaceConfig represents a replacement rule.
type ReplaceConfig struct {
	// Path specifies the relative file path or glob pattern to modify.
	Path string `yaml:"path"`

	// Original specifies the exact string to find.
	Original string `yaml:"original"`

	// Replacement specifies the replacement string.
	Replacement string `yaml:"replacement"`
}

// ReplaceRegexConfig represents a regex replacement rule.
type ReplaceRegexConfig struct {
	// Path specifies the relative file path or glob pattern to modify.
	Path string `yaml:"path"`

	// Pattern specifies the regular expression pattern to find.
	Pattern string `yaml:"pattern"`

	// Replacement specifies the replacement string.
	Replacement string `yaml:"replacement"`
}

// CopyConfig represents a file copy rule.
type CopyConfig struct {
	// Src specifies the source file path relative to the staging directory.
	Src string `yaml:"src"`

	// Dst specifies the destination file path relative to the library root.
	Dst string `yaml:"dst"`
}

// API describes an API to include in a library.
type API struct {
	// Path specifies which googleapis Path to generate from (for generated
	// libraries).
	Path string `yaml:"path,omitempty"`

	// Go contains Go-specific API configuration.
	Go *GoAPI `yaml:"go,omitempty"`

	// Java contains Java-specific API configuration.
	Java *JavaAPI `yaml:"java,omitempty"`

	// Nodejs contains Node.js-specific API configuration.
	Nodejs *NodejsAPI `yaml:"nodejs,omitempty"`

	// PHP contains PHP-specific API configuration.
	PHP *PHPAPI `yaml:"php,omitempty"`
}

// GoDefault defines Go-specific default configuration.
type GoDefault struct {
	// Toolchain is the desired Go toolchain version (e.g., "go1.25.0").
	Toolchain string `yaml:"toolchain,omitempty"`
	// DefaultEnabledGeneratorFeatures lists the generator features enabled by default for all APIs.
	// These default features are appended AFTER any features explicitly declared in individual APIs.
	DefaultEnabledGeneratorFeatures []string `yaml:"default_enabled_generator_features,omitempty"`
}
