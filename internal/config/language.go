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

package config

import (
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	// LanguageUnknown represents an unsupported or unspecified language.
	LanguageUnknown = "unknown"
	// LanguageAll is the identifier for all languages.
	LanguageAll = "all"
	// LanguageCsharp is the language identifier for C#.
	LanguageCsharp = "csharp"
	// LanguageDart is the language identifier for Dart.
	LanguageDart = "dart"
	// LanguageDotnet is the language identifier for .NET.
	LanguageDotnet = "dotnet"
	// LanguageFake is the language identifier for Fakes.
	LanguageFake = "fake"
	// LanguageGo is the language identifier for Go.
	LanguageGo = "go"
	// LanguageJava is the language identifier for Java.
	LanguageJava = "java"
	// LanguageNodejs is the language identifier for Node.js.
	LanguageNodejs = "nodejs"
	// LanguagePhp is the language identifier for PHP.
	LanguagePhp = "php"
	// LanguagePython is the language identifier for Python.
	LanguagePython = "python"
	// LanguageRuby is the language identifier for Ruby.
	LanguageRuby = "ruby"
	// LanguageRust is the language identifier for Rust.
	LanguageRust = "rust"
	// LanguageSwift is the language identifier for Swift.
	LanguageSwift = "swift"
)

// GoModule represents the Go-specific configuration for a library.
type GoModule struct {
	// DeleteGenerationOutputPaths is a list of paths to delete before generation.
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
	// GoAPIs is a list of Go-specific API configurations.
	GoAPIs []*GoAPI `yaml:"go_apis,omitempty"`
	// ModulePathVersion is the version of the Go module path.
	ModulePathVersion string `yaml:"module_path_version,omitempty"`
	// NestedModule is the name of a nested module directory.
	NestedModule string `yaml:"nested_module,omitempty"`
}

// GoAPI represents configuration for a single API within a Go module.
type GoAPI struct {
	// ClientPackage is the package name of the generated client.
	ClientPackage string `yaml:"client_package,omitempty"`
	// DIREGAPIC indicates whether generation uses DIREGAPIC (Discovery REST GAPICs).
	// This is typically false. Used for the GCE (compute) client.
	DIREGAPIC bool `yaml:"diregapic,omitempty"`
	// EnabledGeneratorFeatures provides a mechanism for enabling generator features
	// at the API level.
	EnabledGeneratorFeatures []string `yaml:"enabled_generator_features,omitempty"`
	// ImportPath is the Go import path for the API.
	ImportPath string `yaml:"import_path,omitempty"`
	// NestedProtos is a list of nested proto files.
	NestedProtos []string `yaml:"nested_protos,omitempty"`
	// NoMetadata indicates whether to skip generating gapic_metadata.json.
	// This is typically false.
	NoMetadata bool `yaml:"no_metadata,omitempty"`
	// NoSnippets indicates whether to skip generating snippets.
	// This is typically false.
	NoSnippets bool `yaml:"no_snippets,omitempty"`
	// Path is the source path.
	Path string `yaml:"path,omitempty"`
	// ProtoOnly determines whether to generate a Proto-only client.
	// A proto-only client does not define a service in the proto files.
	ProtoOnly bool `yaml:"proto_only,omitempty"`
	// ProtoPackage is the proto package name.
	ProtoPackage string `yaml:"proto_package,omitempty"`
}

// RustDefault contains Rust-specific default configuration.
type RustDefault struct {
	// PackageDependencies is a list of default package dependencies. These
	// are inherited by all libraries. If a library defines its own
	// package_dependencies, the library-specific ones take precedence over
	// these defaults for dependencies with the same name.
	PackageDependencies []*RustPackageDependency `yaml:"package_dependencies,omitempty"`

	// DisabledRustdocWarnings is a list of rustdoc warnings to disable.
	DisabledRustdocWarnings []string `yaml:"disabled_rustdoc_warnings,omitempty"`

	// GenerateSetterSamples indicates whether to generate setter samples.
	GenerateSetterSamples string `yaml:"generate_setter_samples,omitempty"`

	// GenerateRpcSamples indicates whether to generate RPC samples.
	GenerateRpcSamples string `yaml:"generate_rpc_samples,omitempty"`

	// DetailedTracingAttributes indicates whether to include detailed tracing attributes.
	DetailedTracingAttributes *bool `yaml:"detailed_tracing_attributes,omitempty"`

	// ResourceNameHeuristic indicates whether to apply heuristics to identify and generate resource names.
	ResourceNameHeuristic *bool `yaml:"resource_name_heuristic,omitempty"`
}

// RustModule defines a generation target within a veneer crate.
// Each module specifies what proto source to use, which template to apply,
// and where to output the generated code.
type RustModule struct {
	// DisabledRustdocWarnings specifies rustdoc lints to disable. An empty slice explicitly enables all warnings.
	DisabledRustdocWarnings yaml.StringSlice `yaml:"disabled_rustdoc_warnings,omitempty"`

	// DetailedTracingAttributes indicates whether to include detailed tracing attributes.
	// This overrides the crate-level setting.
	DetailedTracingAttributes *bool `yaml:"detailed_tracing_attributes,omitempty"`

	// DocumentationOverrides contains overrides for element documentation.
	DocumentationOverrides []RustDocumentationOverride `yaml:"documentation_overrides,omitempty"`

	// ExtendGrpcTransport indicates whether the transport stub can be
	// extended (in order to support streams).
	ExtendGrpcTransport bool `yaml:"extend_grpc_transport,omitempty"`

	// GenerateSetterSamples indicates whether to generate setter samples.
	GenerateSetterSamples string `yaml:"generate_setter_samples,omitempty"`

	// GenerateRpcSamples indicates whether to generate RPC samples.
	GenerateRpcSamples string `yaml:"generate_rpc_samples,omitempty"`

	// HasVeneer indicates whether this module has a handwritten wrapper.
	HasVeneer bool `yaml:"has_veneer,omitempty"`

	// IncludedIds is a list of proto IDs to include in generation.
	IncludedIds []string `yaml:"included_ids,omitempty"`

	// IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods.
	IncludeGrpcOnlyMethods bool `yaml:"include_grpc_only_methods,omitempty"`

	// IncludeList is a list of proto files to include (e.g., "date.proto", "expr.proto").
	IncludeList yaml.StringSlice `yaml:"include_list,omitempty"`

	// IncludeStreamingMethods indicates whether to include gRPC streaming
	// methods.
	IncludeStreamingMethods bool `yaml:"include_streaming_methods,omitempty"`

	// InternalBuilders indicates whether generated builders should be internal to the crate.
	InternalBuilders bool `yaml:"internal_builders,omitempty"`

	// ModulePath is the Rust module path for converters
	// (e.g., "crate::generated::gapic::model").
	ModulePath string `yaml:"module_path,omitempty"`

	ModuleRoots map[string]string `yaml:"module_roots,omitempty"`

	// NameOverrides contains codec-level overrides for type and service names.
	NameOverrides string `yaml:"name_overrides,omitempty"`

	// Output is the directory where generated code is written
	// (e.g., "src/storage/src/generated/gapic").
	Output string `yaml:"output"`

	// PostProcessProtos contains code to post-process generated protos.
	PostProcessProtos string `yaml:"post_process_protos,omitempty"`

	// ResourceNameHeuristic indicates whether to apply heuristics to identify and generate resource names.
	// This overrides the crate-level setting.
	ResourceNameHeuristic *bool `yaml:"resource_name_heuristic,omitempty"`

	// RootName is the key for the root directory in the source map.
	// It overrides the default root, googleapis, used by the rust+prost generator.
	RootName string `yaml:"root_name,omitempty"`

	// RoutingRequired indicates whether routing is required.
	RoutingRequired bool `yaml:"routing_required,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// SkippedIds is a list of proto IDs to skip in generation.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// SpecificationFormat overrides the library-level specification format.
	SpecificationFormat string `yaml:"specification_format,omitempty"`

	// APIPath is the proto path to generate from (e.g., "google/storage/v2").
	APIPath string `yaml:"api_path"`

	// Template specifies which generator template to use.
	// Valid values: "grpc-client", "http-client", "prost", "convert-prost", "mod", "storage".
	Template string `yaml:"template"`
}

// RustCrate contains Rust-specific library configuration. It inherits from
// RustDefault, allowing library-specific overrides of global settings.
type RustCrate struct {
	RustDefault `yaml:",inline"`

	// Modules specifies generation targets for veneer crates. Each module
	// defines a source proto path, output location, and template to use.
	Modules []*RustModule `yaml:"modules,omitempty"`

	// PerServiceFeatures enables per-service feature flags.
	PerServiceFeatures bool `yaml:"per_service_features,omitempty"`

	// ModulePath is the module path for the crate.
	ModulePath string `yaml:"module_path,omitempty"`

	// TemplateOverride overrides the default template.
	TemplateOverride string `yaml:"template_override,omitempty"`

	// PackageNameOverride overrides the package name.
	PackageNameOverride string `yaml:"package_name_override,omitempty"`

	// RootName is the root name for the crate.
	RootName string `yaml:"root_name,omitempty"`

	// DefaultFeatures is a list of default features to enable.
	DefaultFeatures []string `yaml:"default_features,omitempty"`

	// IncludeList is a list of proto files to include (e.g., "date.proto", "expr.proto").
	IncludeList []string `yaml:"include_list,omitempty"`

	// IncludedIds is a list of IDs to include.
	IncludedIds []string `yaml:"included_ids,omitempty"`

	// SkippedIds is a list of IDs to skip.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// DisabledClippyWarnings is a list of clippy warnings to disable.
	DisabledClippyWarnings []string `yaml:"disabled_clippy_warnings,omitempty"`

	// HasVeneer indicates whether the crate has a veneer.
	HasVeneer bool `yaml:"has_veneer,omitempty"`

	// RoutingRequired indicates whether routing is required.
	RoutingRequired bool `yaml:"routing_required,omitempty"`

	// IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods.
	IncludeGrpcOnlyMethods bool `yaml:"include_grpc_only_methods,omitempty"`

	// IncludeStreamingMethods indicates whether to include gRPC streaming
	// methods.
	IncludeStreamingMethods bool `yaml:"include_streaming_methods,omitempty"`

	// PostProcessProtos indicates whether to post-process protos.
	PostProcessProtos string `yaml:"post_process_protos,omitempty"`

	// DocumentationOverrides contains overrides for element documentation.
	DocumentationOverrides []RustDocumentationOverride `yaml:"documentation_overrides,omitempty"`

	// PaginationOverrides contains overrides for pagination configuration.
	PaginationOverrides []RustPaginationOverride `yaml:"pagination_overrides,omitempty"`

	// NameOverrides contains codec-level overrides for type and service names.
	NameOverrides string `yaml:"name_overrides,omitempty"`

	// Discovery contains discovery-specific configuration for LRO polling.
	Discovery *RustDiscovery `yaml:"discovery,omitempty"`

	// QuickstartServiceOverride overrides the default heuristically selected service for the package-level quickstart.
	QuickstartServiceOverride string `yaml:"quickstart_service_override,omitempty"`
}

// RustPackageDependency represents a package dependency configuration.
type RustPackageDependency struct {
	// Name is the dependency name. It is listed first so it appears at the top
	// of each dependency entry in YAML.
	Name string `yaml:"name"`

	// Ignore prevents this package from being mapped to an external crate.
	// When true, references to this package stay as `crate::` instead of being
	// mapped to the external crate name. This is used for self-referencing
	// packages like location and longrunning.
	Ignore bool `yaml:"ignore,omitempty"`

	// Package is the package name.
	Package string `yaml:"package"`

	// Source is the dependency source.
	Source string `yaml:"source,omitempty"`

	// Feature is the feature name for the dependency.
	Feature string `yaml:"feature,omitempty"`

	// ForceUsed forces the dependency to be used even if not referenced.
	ForceUsed bool `yaml:"force_used,omitempty"`

	// UsedIf specifies a condition for when the dependency is used.
	UsedIf string `yaml:"used_if,omitempty"`
}

// RustDocumentationOverride represents a documentation override for a specific element.
type RustDocumentationOverride struct {
	// ID is the fully qualified element ID (e.g., .google.cloud.dialogflow.v2.Message.field).
	ID string `yaml:"id"`

	// Match is the text to match in the documentation.
	Match string `yaml:"match"`

	// Replace is the replacement text.
	Replace string `yaml:"replace"`
}

// RustPaginationOverride represents a pagination override for a specific method.
type RustPaginationOverride struct {
	// ID is the fully qualified method ID (e.g., .google.cloud.sql.v1.Service.Method).
	ID string `yaml:"id"`

	// ItemField is the name of the field used for items.
	ItemField string `yaml:"item_field"`
}

// RustDiscovery contains discovery-specific configuration for LRO polling.
type RustDiscovery struct {
	// OperationID is the ID of the LRO operation type (e.g., ".google.cloud.compute.v1.Operation").
	OperationID string `yaml:"operation_id"`

	// Pollers is a list of LRO polling configurations.
	Pollers []RustPoller `yaml:"pollers,omitempty"`
}

// RustPoller defines how to find a suitable poller RPC for discovery APIs.
type RustPoller struct {
	// Prefix is an acceptable prefix for the URL path (e.g., "compute/v1/projects/{project}/zones/{zone}").
	Prefix string `yaml:"prefix"`

	// MethodID is the corresponding method ID (e.g., ".google.cloud.compute.v1.zoneOperations.get").
	MethodID string `yaml:"method_id"`
}

// PythonPackage contains Python-specific library configuration. It inherits
// from PythonDefault, allowing library-specific overrides of global settings.
type PythonPackage struct {
	PythonDefault `yaml:",inline"`

	// OptArgsByAPI contains additional options passed to the generator.
	// In each entry, the key is the API path and the value is the list of
	// options to pass when generating that API.
	// Example: {"google/cloud/secrets/v1beta": ["python-gapic-name=secretmanager"]}
	OptArgsByAPI map[string][]string `yaml:"opt_args_by_api,omitempty"`

	// ProtoOnlyAPIs contains the list of API paths which are proto-only, so
	// should use regular protoc Python generation instead of GAPIC.
	ProtoOnlyAPIs []string `yaml:"proto_only_apis,omitempty"`

	// NamePrettyOverride allows the "name_pretty" field in .repo-metadata.json
	// to be overridden, to reduce diffs while migrating.
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove this
	// field.
	NamePrettyOverride string `yaml:"name_pretty_override,omitempty"`

	// ProductDocumentationOverride allows the "product_documentation" field in
	// .repo-metadata.json to be overridden, to reduce diffs while migrating.
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove this
	// field.
	ProductDocumentationOverride string `yaml:"product_documentation_override,omitempty"`

	// APIShortnameOverride allows the "api_shortname" field in
	// .repo-metadata.json to be overridden, to reduce diffs while migrating.
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove this
	// field.
	APIShortnameOverride string `yaml:"api_shortname_override,omitempty"`

	// APIIDOverride allows the "api_id" field in
	// .repo-metadata.json to be overridden, to reduce diffs while migrating.
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove this
	// field.
	APIIDOverride string `yaml:"api_id_override,omitempty"`

	// ClientDocumentationOverride allows the client_documentation field in
	// .repo-metadata.json to be overridden from the default that's inferred.
	// TODO(https://github.com/googleapis/librarian/issues/4175): reduce uses
	// of this field to only cases where it's really needed.
	ClientDocumentationOverride string `yaml:"client_documentation_override,omitempty"`

	// IssueTrackerOverride allows the issue_tracker field in
	// .repo-metadata.json to be overridden, to reduce diffs while migrating.
	// TODO(https://github.com/googleapis/librarian/issues/4175): remove this
	// field.
	IssueTrackerOverride string `yaml:"issue_tracker_override,omitempty"`

	// MetadataNameOverride allows the name in .repo-metadata.json (which is
	// also used as part of the client documentation URI) to be overridden. By
	// default, it's the package name, but older packages use the API short name
	// instead.
	MetadataNameOverride string `yaml:"metadata_name_override,omitempty"`

	// DefaultVersion is the default version of the API to use. When omitted,
	// the version in the first API path is used.
	DefaultVersion string `yaml:"default_version,omitempty"`

	// SkipReadmeCopy prevents generation from copying README.rst from the root
	// directory to the docs directory.
	// TODO(https://github.com/googleapis/librarian/issues/4738): revisit
	// whether or not this field should exist after migration.
	SkipReadmeCopy bool `yaml:"skip_readme_copy,omitempty"`
}

// PythonDefault contains Python-specific default configuration.
type PythonDefault struct {
	// CommonGAPICPaths contains paths which are generated for any package
	// containing a GAPIC API. These are relative to the package's output
	// directory, and the string "{neutral-source}" is replaced with the path
	// to the version-neutral source code (e.g. "google/cloud/run"). If a
	// library defines its own common_gapic_paths, they will be appended to
	// the defaults.
	CommonGAPICPaths []string `yaml:"common_gapic_paths,omitempty"`

	// LibraryType is the type to emit in .repo-metadata.json.
	LibraryType string `yaml:"library_type,omitempty"`
}

// DartPackage contains Dart-specific library configuration.
type DartPackage struct {
	// APIKeysEnvironmentVariables is a comma-separated list of environment variable names
	// that can contain API keys (e.g., "GOOGLE_API_KEY,GEMINI_API_KEY").
	APIKeysEnvironmentVariables string `yaml:"api_keys_environment_variables,omitempty"`

	// Dependencies is a comma-separated list of dependencies.
	Dependencies string `yaml:"dependencies,omitempty"`

	// DevDependencies is a comma-separated list of development dependencies.
	DevDependencies string `yaml:"dev_dependencies,omitempty"`

	// ExtraImports is additional imports to include in the generated library.
	ExtraImports string `yaml:"extra_imports,omitempty"`

	// IncludeList is a list of proto files to include (e.g., "date.proto", "expr.proto").
	IncludeList []string `yaml:"include_list,omitempty"`

	// IssueTrackerURL is the URL for the issue tracker.
	IssueTrackerURL string `yaml:"issue_tracker_url,omitempty"`

	// LibraryPathOverride overrides the library path.
	LibraryPathOverride string `yaml:"library_path_override,omitempty"`

	// NameOverride overrides the package name
	NameOverride string `yaml:"name_override,omitempty"`

	// Packages maps Dart package names to version constraints.
	// Keys are in the format "package:googleapis_auth" and values are version strings like "^2.0.0".
	// These are merged with default settings, with library settings taking precedence.
	Packages map[string]string `yaml:"packages,omitempty"`

	// PartFile is the path to a part file to include in the generated library.
	PartFile string `yaml:"part_file,omitempty"`

	// Prefixes maps protobuf package names to Dart import prefixes.
	// Keys are in the format "prefix:google.protobuf" and values are the prefix names.
	// These are merged with default settings, with library settings taking precedence.
	Prefixes map[string]string `yaml:"prefixes,omitempty"`

	// Protos maps protobuf package names to Dart import paths.
	// Keys are in the format "proto:google.api" and values are import paths like "package:google_cloud_api/api.dart".
	// These are merged with default settings, with library settings taking precedence.
	Protos map[string]string `yaml:"protos,omitempty"`

	// ReadmeAfterTitleText is text to insert in the README after the title.
	ReadmeAfterTitleText string `yaml:"readme_after_title_text,omitempty"`

	// ReadmeQuickstartText is text to use for the quickstart section in the README.
	ReadmeQuickstartText string `yaml:"readme_quickstart_text,omitempty"`

	// RepositoryURL is the URL to the repository for this package.
	RepositoryURL string `yaml:"repository_url,omitempty"`

	// TitleOverride overrides the API title.
	TitleOverride string `yaml:"title_override,omitempty"`

	// Version is the version of the dart package.
	Version string `yaml:"version,omitempty"`
}

// JavaModule contains Java-specific library configuration.
// TODO(https://github.com/googleapis/librarian/issues/4130):
// add fill defaults for fields with default.
type JavaModule struct {
	// APIIDOverride is the ID of the API (e.g., "pubsub.googleapis.com"),
	// allows the "api_id" field in .repo-metadata.json to be overridden.
	// Defaults to "{library.api_shortname}.googleapis.com".
	APIIDOverride string `yaml:"api_id_override,omitempty"`

	// APIReference is the URL for the API reference documentation.
	APIReference string `yaml:"api_reference,omitempty"`

	// APIDescriptionOverride allows the "api_description" field in
	// .repo-metadata.json to be overridden.
	APIDescriptionOverride string `yaml:"api_description_override,omitempty"`

	// APIShortnameOverride allows the "api_shortname" field in
	// .repo-metadata.json to be overridden.
	APIShortnameOverride string `yaml:"api_shortname_override,omitempty"`

	// ClientDocumentationOverride allows the "client_documentation" field in
	// .repo-metadata.json to be overridden.
	ClientDocumentationOverride string `yaml:"client_documentation_override,omitempty"`

	// NonCloudAPI indicates whether the API is NOT a Google Cloud API.
	// Defaults to false.
	NonCloudAPI bool `yaml:"non_cloud_api,omitempty"`

	// CodeownerTeam is the GitHub team that owns the code.
	CodeownerTeam string `yaml:"codeowner_team,omitempty"`

	// DistributionNameOverride allows the "distribution_name" field in
	// .repo-metadata.json to be overridden.
	DistributionNameOverride string `yaml:"distribution_name_override,omitempty"`

	// ExcludedDependencies is a list of dependencies to exclude.
	ExcludedDependencies string `yaml:"excluded_dependencies,omitempty"`

	// ExcludedPOMs is a list of POM files to exclude.
	ExcludedPOMs string `yaml:"excluded_poms,omitempty"`

	// ExtraVersionedModules is a list of extra versioned modules.
	ExtraVersionedModules string `yaml:"extra_versioned_modules,omitempty"`

	// GroupID is the Maven group ID, defaults to "com.google.cloud".
	GroupID string `yaml:"group_id,omitempty"`

	// IssueTrackerOverride allows the "issue_tracker" field in .repo-metadata.json
	// to be overridden.
	IssueTrackerOverride string `yaml:"issue_tracker_override,omitempty"`

	// LibrariesBOMVersion is the version of the libraries-bom to use for Java.
	LibrariesBOMVersion string `yaml:"libraries_bom_version,omitempty"`

	// LibraryTypeOverride allows the "library_type" field in .repo-metadata.json
	// to be overridden.
	LibraryTypeOverride string `yaml:"library_type_override,omitempty"`

	// MinJavaVersion is the minimum Java version required.
	MinJavaVersion int `yaml:"min_java_version,omitempty"`

	// NamePrettyOverride allows the "name_pretty" field in .repo-metadata.json
	// to be overridden.
	NamePrettyOverride string `yaml:"name_pretty_override,omitempty"`

	// JavaAPIs is a list of Java-specific API configurations.
	JavaAPIs []*JavaAPI `yaml:"java_apis,omitempty"`

	// ProductDocumentationOverride allows the "product_documentation" field in
	// .repo-metadata.json to be overridden.
	ProductDocumentationOverride string `yaml:"product_documentation_override,omitempty"`

	// RecommendedPackage is the recommended package name.
	RecommendedPackage string `yaml:"recommended_package,omitempty"`

	// BillingNotRequired indicates whether the API does NOT require billing.
	// This is typically false.
	BillingNotRequired bool `yaml:"billing_not_required,omitempty"`

	// RestDocumentation is the URL for the REST documentation.
	RestDocumentation string `yaml:"rest_documentation,omitempty"`

	// RpcDocumentation is the URL for the RPC documentation.
	RpcDocumentation string `yaml:"rpc_documentation,omitempty"`
}

// JavaAPI represents configuration for a single API within a Java module.
type JavaAPI struct {
	// Path is the source path.
	Path string `yaml:"path,omitempty"`

	// AdditionalProtos is a list of additional proto files to include in generation.
	AdditionalProtos []string `yaml:"additional_protos,omitempty"`

	// ExcludedProtos is a list of proto files to exclude from generation.
	// It expects the full path starting from the root of the googleapis
	// directory (e.g., "google/cloud/aiplatform/v1/schema/io_format.proto").
	ExcludedProtos []string `yaml:"excluded_protos,omitempty"`

	// GapicArtifactIDOverride overrides the artifact ID for the GAPIC module.
	// It determines the module's directory name and is used to derive proto
	// and gRPC artifact IDs if they are not explicitly overridden.
	GapicArtifactIDOverride string `yaml:"gapic_artifact_id_override,omitempty"`

	// GRPCArtifactIDOverride overrides the artifact ID for the gRPC module.
	// The artifact ID is also used as the name for the module's directory.
	GRPCArtifactIDOverride string `yaml:"grpc_artifact_id_override,omitempty"`

	// ProtoArtifactIDOverride overrides the artifact ID for the proto module.
	// The artifact ID is also used as the name for the module's directory.
	ProtoArtifactIDOverride string `yaml:"proto_artifact_id_override,omitempty"`

	// ProtoOnly determines whether to generate a Proto-only client.
	// A proto-only client does not define a service in the proto files.
	ProtoOnly bool `yaml:"proto_only,omitempty"`

	// Samples determines whether to generate samples for the API,
	// default is true when omitted.
	Samples *bool `yaml:"samples,omitempty"`
}

// DotnetPackage contains .NET-specific library configuration.
type DotnetPackage struct {
	// AdditionalServiceDescriptors is a list of extra service descriptors to include.
	AdditionalServiceDescriptors []string `yaml:"additional_service_descriptors,omitempty"`

	// Csproj contains configuration for .csproj file generation and overrides.
	Csproj *DotnetCsproj `yaml:"csproj,omitempty"`

	// Dependencies maps NuGet package IDs to version strings.
	Dependencies map[string]string `yaml:"dependencies,omitempty"`

	// Generator overrides the default generator (e.g., "proto").
	Generator string `yaml:"generator,omitempty"`

	// PackageGroup lists packages that must be released together.
	PackageGroup []string `yaml:"package_group,omitempty"`

	// Postgeneration contains post-generation shell commands or extra protos.
	Postgeneration []*DotnetPostgeneration `yaml:"postgeneration,omitempty"`

	// Pregeneration contains declarative proto mutations.
	Pregeneration []*DotnetPregeneration `yaml:"pregeneration,omitempty"`
}

// DotnetPregeneration represents a declarative proto mutation.
type DotnetPregeneration struct {
	// RenameMessage renames a message.
	RenameMessage *DotnetRenameMessage `yaml:"rename_message,omitempty"`

	// RemoveField removes a field from a message.
	RemoveField *DotnetRemoveField `yaml:"remove_field,omitempty"`

	// RenameRPC renames an RPC.
	RenameRPC *DotnetRenameRPC `yaml:"rename_rpc,omitempty"`
}

// DotnetRenameMessage contains rename message configuration.
type DotnetRenameMessage struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// DotnetRemoveField contains remove field configuration.
type DotnetRemoveField struct {
	Message string `yaml:"message"`
	Field   string `yaml:"field"`
}

// DotnetRenameRPC contains rename RPC configuration.
type DotnetRenameRPC struct {
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	WireName string `yaml:"wire_name,omitempty"`
}

// DotnetPostgeneration represents a post-generation action.
type DotnetPostgeneration struct {
	// Run is a shell command to execute.
	Run string `yaml:"run,omitempty"`

	// ExtraProto is an extra proto file to compile.
	ExtraProto string `yaml:"extra_proto,omitempty"`
}

// DotnetCsproj contains configuration for .csproj file generation.
type DotnetCsproj struct {
	// Snippets contains XML fragments for .csproj files.
	Snippets *DotnetCsprojSnippets `yaml:"snippets,omitempty"`

	// IntegrationTests contains configuration for integration test projects.
	IntegrationTests *DotnetCsprojSnippets `yaml:"integration_tests,omitempty"`
}

// DotnetCsprojSnippets contains XML fragments to be merged into .csproj files.
type DotnetCsprojSnippets struct {
	// EmbeddedResources is a list of glob patterns for embedded resources.
	EmbeddedResources []string `yaml:"embedded_resources,omitempty"`
}

// NodejsPackage contains Node.js-specific library configuration.
type NodejsPackage struct {
	// AdditionalProtos is a list of additional proto files to include in generation.
	// This can be overridden at the API level.
	AdditionalProtos []string `yaml:"additional_protos,omitempty"`

	// BundleConfig is the path to a GAPIC bundle config file.
	BundleConfig string `yaml:"bundle_config,omitempty"`

	// Dependencies maps npm package names to version constraints.
	Dependencies map[string]string `yaml:"dependencies,omitempty"`

	// ExtraProtocParameters is a list of extra parameters to pass to protoc.
	ExtraProtocParameters []string `yaml:"extra_protoc_parameters,omitempty"`

	// HandwrittenLayer indicates the library has a handwritten layer on top
	// of the generated code.
	HandwrittenLayer bool `yaml:"handwritten_layer,omitempty"`

	// MainService is the name of the main service for libraries with a
	// handwritten layer.
	MainService string `yaml:"main_service,omitempty"`

	// Mixins controls mixin behavior (e.g., "none" to disable).
	Mixins string `yaml:"mixins,omitempty"`

	// NodejsAPIs is a list of Node.js-specific API configurations.
	NodejsAPIs []*NodejsAPI `yaml:"nodejs_apis,omitempty"`

	// PackageName is the npm package name (e.g., "@google-cloud/access-approval").
	PackageName string `yaml:"package_name,omitempty"`
}

// NodejsAPI represents configuration for a single API within a Node.js package.
type NodejsAPI struct {
	// AdditionalProtos is a list of additional proto files to include in generation.
	AdditionalProtos []string `yaml:"additional_protos,omitempty"`

	// Path is the source path.
	Path string `yaml:"path,omitempty"`
}
