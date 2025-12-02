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

// RustDefault contains Rust-specific default configuration.
type RustDefault struct {
	// PackageDependencies is a list of default package dependencies.
	PackageDependencies []*RustPackageDependency `yaml:"package_dependencies,omitempty"`

	// DisabledRustdocWarnings is a list of rustdoc warnings to disable.
	DisabledRustdocWarnings []string `yaml:"disabled_rustdoc_warnings,omitempty"`
}

// RustCrate contains Rust-specific library configuration.
type RustCrate struct {
	RustDefault `yaml:",inline"`

	// PerServiceFeatures enables per-service feature flags.
	PerServiceFeatures bool `yaml:"per_service_features,omitempty"`

	// ModulePath is the module path for the crate.
	ModulePath string `yaml:"module_path,omitempty"`

	// TemplateOverride overrides the default template.
	TemplateOverride string `yaml:"template_override,omitempty"`

	// TitleOverride overrides the crate title.
	TitleOverride string `yaml:"title_override,omitempty"`

	// DescriptionOverride overrides the crate description.
	DescriptionOverride string `yaml:"description_override,omitempty"`

	// PackageNameOverride overrides the package name.
	PackageNameOverride string `yaml:"package_name_override,omitempty"`

	// RootName is the root name for the crate.
	RootName string `yaml:"root_name,omitempty"`

	// Roots is a list of root names.
	Roots []string `yaml:"roots,omitempty"`

	// DefaultFeatures is a list of default features to enable.
	DefaultFeatures []string `yaml:"default_features,omitempty"`

	// ExtraModules is a list of extra modules to include.
	ExtraModules []string `yaml:"extra_modules,omitempty"`

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
