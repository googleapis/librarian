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

// GoModule represents the Go-specific configuration for a library.
type GoModule struct {
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
	GoAPIs                      []*GoAPI `yaml:"go_apis,omitempty"`
	ModulePathVersion           string   `yaml:"module_path_version,omitempty"`
}

// GoAPI represents configuration for a single API channel within a Go module.
type GoAPI struct {
	Path            string   `yaml:"path,omitempty"`
	ClientDirectory string   `yaml:"client_directory,omitempty"`
	DisableGAPIC    bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos    []string `yaml:"nested_protos,omitempty"`
	ProtoPackage    string   `yaml:"proto_package,omitempty"`
}

// RustDefault contains Rust-specific default configuration.
type RustDefault struct {
	// PackageDependencies is a list of default package dependencies.
	PackageDependencies []*RustPackageDependency `yaml:"package_dependencies,omitempty"`

	// DisabledRustdocWarnings is a list of rustdoc warnings to disable.
	DisabledRustdocWarnings []string `yaml:"disabled_rustdoc_warnings,omitempty"`
}

// RustModule defines a generation target within a veneer crate.
// Each module specifies what proto source to use, which template to apply,
// and where to output the generated code.
type RustModule struct {
	// GenerateSetterSamples indicates whether to generate setter samples.
	GenerateSetterSamples bool `yaml:"generate_setter_samples,omitempty"`

	// HasVeneer indicates whether this module has a hand-written wrapper.
	HasVeneer bool `yaml:"has_veneer,omitempty"`

	// IncludedIds is a list of proto IDs to include in generation.
	IncludedIds []string `yaml:"included_ids,omitempty"`

	// IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods.
	IncludeGrpcOnlyMethods bool `yaml:"include_grpc_only_methods,omitempty"`

	// IncludeList is a list of proto files to include (e.g., "date.proto,expr.proto").
	IncludeList string `yaml:"include_list,omitempty"`

	// ModulePath is the Rust module path for converters
	// (e.g., "crate::generated::gapic::model").
	ModulePath string `yaml:"module_path,omitempty"`

	// NameOverrides contains codec-level overrides for type and service names.
	NameOverrides string `yaml:"name_overrides,omitempty"`

	// Output is the directory where generated code is written
	// (e.g., "src/storage/src/generated/gapic").
	Output string `yaml:"output"`

	// PostProcessProtos contains code to post-process generated protos.
	PostProcessProtos string `yaml:"post_process_protos,omitempty"`

	// Roots is a list of root names to be used as source directories.
	Roots []string `yaml:"roots,omitempty"`

	// RoutingRequired indicates whether routing is required.
	RoutingRequired bool `yaml:"routing_required,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// SkippedIds is a list of proto IDs to skip in generation.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// Source is the proto path to generate from (e.g., "google/storage/v2").
	Source string `yaml:"source"`

	// Template specifies which generator template to use.
	// Valid values: "grpc-client", "http-client", "prost", "convert-prost", "mod".
	Template string `yaml:"template"`

	// TitleOverride overrides the crate title.
	TitleOverride string `yaml:"title_override,omitempty"`
}

// RustCrate contains Rust-specific library configuration.
type RustCrate struct {
	RustDefault `yaml:",inline"`

	// Modules specifies generation targets for veneer crates. Each module
	// defines a source proto path, output location, and template to use.
	// This is only used when the library has veneer: true.
	Modules []*RustModule `yaml:"modules,omitempty"`

	// PerServiceFeatures enables per-service feature flags.
	PerServiceFeatures bool `yaml:"per_service_features,omitempty"`

	// ModulePath is the module path for the crate.
	ModulePath string `yaml:"module_path,omitempty"`

	// TemplateOverride overrides the default template.
	TemplateOverride string `yaml:"template_override,omitempty"`

	// TitleOverride overrides the crate title.
	TitleOverride string `yaml:"title_override,omitempty"`

	// PackageNameOverride overrides the package name.
	PackageNameOverride string `yaml:"package_name_override,omitempty"`

	// RootName is the root name for the crate.
	RootName string `yaml:"root_name,omitempty"`

	// Roots is a list of root names.
	Roots []string `yaml:"roots,omitempty"`

	// DefaultFeatures is a list of default features to enable.
	DefaultFeatures []string `yaml:"default_features,omitempty"`

	// IncludeList is a list of items to include.
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

	// GenerateSetterSamples indicates whether to generate setter samples.
	GenerateSetterSamples bool `yaml:"generate_setter_samples,omitempty"`

	// GenerateRpcSamples indicates whether to generate RPC samples.
	GenerateRpcSamples bool `yaml:"generate_rpc_samples,omitempty"`

	// PostProcessProtos indicates whether to post-process protos.
	PostProcessProtos string `yaml:"post_process_protos,omitempty"`

	// DetailedTracingAttributes indicates whether to include detailed tracing attributes.
	DetailedTracingAttributes bool `yaml:"detailed_tracing_attributes,omitempty"`

	// DocumentationOverrides contains overrides for element documentation.
	DocumentationOverrides []RustDocumentationOverride `yaml:"documentation_overrides,omitempty"`

	// PaginationOverrides contains overrides for pagination configuration.
	PaginationOverrides []RustPaginationOverride `yaml:"pagination_overrides,omitempty"`

	// NameOverrides contains codec-level overrides for type and service names.
	NameOverrides string `yaml:"name_overrides,omitempty"`

	// Discovery contains discovery-specific configuration for LRO polling.
	Discovery *RustDiscovery `yaml:"discovery,omitempty"`
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

// PythonPackage contains Python-specific library configuration.
type PythonPackage struct {
	// OptArgs contains additional options passed to the generator, where
	// the options are common to all channels.
	// Example: ["warehouse-package-name=google-cloud-batch"]
	OptArgs []string `yaml:"opt_args,omitempty"`

	// OptArgsByChannel contains additional options passed to the generator,
	// where the options vary by channel. In each entry, the key is the channel
	// (API path) and the value is the list of options to pass when generating
	// that API channel.
	// Example: {"google/cloud/secrets/v1beta": ["python-gapic-name=secretmanager"]}
	OptArgsByChannel map[string][]string `yaml:"opt_args_by_channel,omitempty"`
}
