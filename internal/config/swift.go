// Copyright 2026 Google LLC
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

// SwiftDefault contains the configuration shared by all Swift libraries.
type SwiftDefault struct {
	// Dependencies is a list of package dependencies.
	Dependencies []SwiftDependency `yaml:"dependencies,omitempty"`

	// The default version for new libraries.
	DefaultVersion string `yaml:"default_version,omitempty"`
}

// SwiftPackage contains Swift-specific configuration for a Swift library.
//
// It inherits from SwiftDefault, allowing library-specific overrides of global settings.
type SwiftPackage struct {
	SwiftDefault `yaml:",inline"`

	// LibraryNameOverride overrides the default library name.
	//
	// In Swift, each GAPIC package consists of a single product (the library),
	// which contains a single target and module name. For example, the package for
	// the google/cloud/secretmanager/v1 API is called google-cloud-secretmanager-v1, and
	// contains a single product: `GoogleCloudSecretManagerV1`, which in turn contains a single target and module of the same name.
	//
	// To use the library applications use this import:
	//
	// ```
	// import GoogleCloudSecretManagerV1
	// ```
	//
	// Normally the name is derived from:
	// - If the Protobuf namespace overrides for PHP, Ruby, and C# are consistent, sidekick uses this name.
	// - Otherwise, the name implied by the Protobuf package
	// - Or the package set in the service config yaml file
	LibraryNameOverride string `yaml:"library_name_override,omitempty"`

	// IncludeList is a subset of proto files under the target API path to
	// include (e.g., ["date.proto", "expr.proto"]).
	IncludeList []string `yaml:"include_list,omitempty"`

	// SkippedIds is a list of proto IDs to skip in generation for the package.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// Modules specifies generation targets for veneers and test packages.
	//
	// Each module defines a source proto path, and output location.
	Modules []*SwiftModule `yaml:"modules,omitempty"`

	// PackageNameOverride overrides the package name.
	//
	// This may be useful if the protobuf package lacks the necessary prefixes,
	// e.g. `grafeas.v1` may be published as `google-grafeas-v1` to match the
	// other packages.
	PackageNameOverride string `yaml:"package_name_override,omitempty"`

	// PerServiceTraits enables per-service compile-time flags.
	PerServiceTraits bool `yaml:"per_service_traits,omitempty"`

	// DefaultTraits is a list of compile-time traits enabled by default.
	DefaultTraits []string `yaml:"default_traits,omitempty"`

	// Discovery contains discovery-specific configuration for LRO polling.
	Discovery *SwiftDiscovery `yaml:"discovery,omitempty"`
}

// SwiftDependency represents a dependency in Swift Package Manager.
type SwiftDependency struct {
	// Name is the module imported from the dependency.
	//
	// Examples:
	// - to import `Logging` from the `swift-log` package, create a dependency:
	//     {name: "Logging", version: "1.14.0", url: "https://github.com/apple/swift-log"},
	// - to import `GoogleIamV1` from the `google-iam-v1` package.
	//     {name: "GoogleIamV1", path: "generated/google-iam-v1"}
	Name string `yaml:"name"`
	// Path configures the path for local (to the monorepo) packages.
	//
	// For example, the authentication package definition will set this to `packages/auth`, which
	// would generate the following snippet in the `Package.swift` files:
	//
	// ```
	// .package(path: "../../packages/auth")
	// ```
	Path string `yaml:"path,omitempty"`
	// URL configures the `url:` parameter in the package definition.
	//
	// For example, `https://github.com/apple/swift-protobuf` would generate the following snippet in
	// the `Package.swift` files:
	//
	// ```
	// .package(url: "https://github.com/apple/swift-protobuf")
	// ```
	URL string `yaml:"url,omitempty"`
	// Version configures the minimum version for external package definitions.
	//
	// For example, if the `swift-protobuf` package used `1.36.1`, then the codec would generate the
	// following snippet in the `Package.swift` files:
	//
	// ```
	// .package(url: "https://github.com/apple/swift-protobuf", from: "1.36.1")
	// ```
	Version string `yaml:"version,omitempty"`
	// RequiredByServices is true if this dependency is required by packages with services.
	//
	// This will be set for the `gax` library and the `auth` library. Maybe more if we split the HTTP
	// and gRPC clients into separate libraries.
	RequiredByServices bool `yaml:"required_by_services,omitempty"`
	// ApiPackage is the name of the API package provided by this library.
	//
	// In Swift a package contains at most one channel for one API. For packages that implement an
	// API, this field contains the name of the package in the specification language of that API.
	// At the moment this is only used by Protobuf-based APIs, as OpenAPI and discovery doc APIs are
	// self-contained.
	//
	// Note that some packages, for example `auth` and `gax`, do not implement APIs. This field is
	// empty for such libraries.
	//
	// Examples:
	// - The `GoogleCloudWkt` package will set this to `google.cloud.protobuf`.
	// - The `GoogleCloudLocation` package will set this to `google.cloud.location`.
	ApiPackage string `yaml:"api_package,omitempty"`
}

// SwiftModule defines a generation target within a larger crate. Typically a veneer, but sometimes also test targets.
//
// Each module specifies what proto source to use, and where to output the generated code.
type SwiftModule struct {
	// Output is the directory where generated code is written (e.g., "Tests/ProtoJSON/generated").
	Output string `yaml:"output"`

	// ServiceConfig is the path to the service config file (e.g., "google/storage/control/v2/storage_v2.yaml").
	ServiceConfig string `yaml:"service_config,omitempty"`

	// APIPath is the proto path to generate from (e.g., "google/storage/v2").
	APIPath string `yaml:"api_path"`

	// ModuleType is the type of module to generate (e.g., "swift-protobuf", "convert-swift", or empty/"default" for standard GAPIC).
	ModuleType string `yaml:"module_type,omitempty"`

	// IncludeList is a subset of proto files under the target API path to include.
	// This is typically reserved for special cases to avoid generating unused/dead code.
	// For example, in Storage we need Protobuf gencode for a subset of the protos in the google/type
	// directory. This code is private to the package (google-cloud-storage in Rust, GoogleCloudStorage
	// in Swift). All other files in google/type would be dead code.
	IncludeList []string `yaml:"include_list,omitempty"`

	// SkippedIds is a list of proto IDs to skip in generation for this module.
	SkippedIds []string `yaml:"skipped_ids,omitempty"`

	// ModulePath is the module import path or target containing stubs (used by convert-swift).
	ModulePath string `yaml:"module_path,omitempty"`
}

// SwiftDiscovery contains discovery-specific configuration for LRO polling.
type SwiftDiscovery = CommonDiscovery

// SwiftPoller defines how to find a suitable poller RPC for discovery APIs.
type SwiftPoller = CommonPoller
