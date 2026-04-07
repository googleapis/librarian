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
	// SourcePackage is not empty, this is the name of the Protobuf package provided by this library.
	//
	// Examples:
	// - The `GoogleCloudWkt` package will set this to `google.cloud.protobuf`.
	// - The `GoogleCloudLocation` package will set this to `google.cloud.location`.
	SourcePackage string `yaml:"source_package,omitempty"`
}
