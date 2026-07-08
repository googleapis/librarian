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
}

// SwiftPackage contains Swift-specific configuration for a Swift library.
//
// It inherits from SwiftDefault, allowing library-specific overrides of global settings.
type SwiftPackage struct {
	SwiftDefault `yaml:",inline"`

	// IncludeList is a subset of proto files under the target API path to
	// include (e.g., ["date.proto", "expr.proto"]).
	IncludeList []string `yaml:"include_list,omitempty"`

	// Modules specifies generation targets for veneers and test packages.
	//
	// Each module defines a source proto path, and output location.
	Modules []*SwiftModule `yaml:"modules,omitempty"`

	// PerServiceTraits enables per-service compile-time flags.
	PerServiceTraits bool `yaml:"per_service_traits,omitempty"`

	// DefaultTraits is a list of compile-time traits enabled by default.
	DefaultTraits []string `yaml:"default_traits,omitempty"`

	// Discovery contains discovery-specific configuration for LRO polling.
	Discovery *SwiftDiscovery `yaml:"discovery,omitempty"`
}

// SwiftDependency represents a dependency in Swift Package Manager.
type SwiftDependency struct {
	// Name is an identifier for the package within the project.
	//
	// For example, `swift-protobuf`. This is usually the last component of the path or the URL.
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
	// Version configures the minimum version for exaternal package definitions.
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

	// APIPath is the proto path to generate from (e.g., "google/storage/v2").
	APIPath string `yaml:"api_path"`

	// ModuleType is the type of module to generate (e.g., "swift-protobuf", "convert-swift", or empty for standard GAPIC).
	ModuleType string `yaml:"module_type,omitempty"`

	// IncludeList is a subset of proto files under the target API path to include.
	// This is typically reserved for special cases (e.g., resolving dependency issues or
	// compiling subsets of shared proto namespaces) and should not be used for standard libraries.
	IncludeList []string `yaml:"include_list,omitempty"`

	// ModulePath is the module import path or target containing stubs (used by convert-swift).
	ModulePath string `yaml:"module_path,omitempty"`
}

// SwiftDiscovery contains discovery-specific configuration for LRO polling.
type SwiftDiscovery = CommonDiscovery

// SwiftPoller defines how to find a suitable poller RPC for discovery APIs.
type SwiftPoller = CommonPoller
