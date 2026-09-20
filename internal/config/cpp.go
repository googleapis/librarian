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

// CppDefault contains C++ default configuration options shared across libraries.
type CppDefault struct {
	// DefaultVersion is the default library version.
	DefaultVersion string `yaml:"default_version,omitempty"`
}

// CppLibrary contains C++ library-specific configuration for a library.
type CppLibrary struct {
	// SourceRoot is the optional root directory of the proto sources.
	SourceRoot string `yaml:"source_root,omitempty"`

	// ProductPath is the relative path of the generated versioned library.
	// Defaults to library.Output if omitted.
	ProductPath string `yaml:"product_path,omitempty"`

	// ForwardingProductPath is the relative directory for top-level forwarding headers.
	ForwardingProductPath string `yaml:"forwarding_product_path,omitempty"`

	// ServiceEndpointEnvVar is the environment variable used to override the service endpoint.
	ServiceEndpointEnvVar string `yaml:"service_endpoint_env_var,omitempty"`

	// EmulatorEndpointEnvVar is the environment variable used to override the emulator endpoint.
	EmulatorEndpointEnvVar string `yaml:"emulator_endpoint_env_var,omitempty"`

	// GenerateRestTransport specifies whether to generate REST transport code.
	GenerateRestTransport bool `yaml:"generate_rest_transport,omitempty"`

	// GenerateGrpcTransport specifies whether to generate gRPC transport code.
	GenerateGrpcTransport *bool `yaml:"generate_grpc_transport,omitempty"`

	// EndpointLocationStyle controls endpoint location derivation style.
	EndpointLocationStyle string `yaml:"endpoint_location_style,omitempty"`

	// BackwardsCompatibilityNamespace controls generation of backward compatibility namespace alias.
	BackwardsCompatibilityNamespace bool `yaml:"backwards_compatibility_namespace_alias,omitempty"`

	// OmittedRPCs lists RPC method names to omit from code generation.
	OmittedRPCs []string `yaml:"omitted_rpcs,omitempty"`

	// GenAsyncRPCs lists RPC method names for which asynchronous client methods are generated.
	GenAsyncRPCs []string `yaml:"gen_async_rpcs,omitempty"`

	// OmittedServices lists service names to omit from code generation.
	OmittedServices []string `yaml:"omitted_services,omitempty"`

	// RetryableStatusCodes lists gRPC status codes that are treated as retryable by default.
	RetryableStatusCodes []string `yaml:"retryable_status_codes,omitempty"`

	// IdempotencyOverrides lists custom idempotency settings for specific RPCs.
	IdempotencyOverrides []IdempotencyRule `yaml:"idempotency_overrides,omitempty"`

	// GenerateRoundRobinDecorator indicates whether to generate round-robin stubs.
	GenerateRoundRobinDecorator bool `yaml:"generate_round_robin_decorator,omitempty"`

	// OmitClient indicates whether to omit generating client class.
	OmitClient bool `yaml:"omit_client,omitempty"`

	// OmitConnection indicates whether to omit generating connection class.
	OmitConnection bool `yaml:"omit_connection,omitempty"`

	// OmitStubFactory indicates whether to omit generating stub factory functions.
	OmitStubFactory bool `yaml:"omit_stub_factory,omitempty"`

	// AdditionalProtoFiles lists extra proto files to include during parsing.
	AdditionalProtoFiles []string `yaml:"additional_proto_files,omitempty"`

	// OverrideServiceConfigYAMLName specifies a path to a service config yaml override.
	OverrideServiceConfigYAMLName string `yaml:"override_service_config_yaml_name,omitempty"`

	// InitialCopyrightYear specifies the initial copyright year to preserve in headers.
	InitialCopyrightYear string `yaml:"initial_copyright_year,omitempty"`

	// OmitRepoMetadata indicates whether to skip emitting .repo-metadata.json.
	OmitRepoMetadata bool `yaml:"omit_repo_metadata,omitempty"`

	// ServiceNameMapping maps original service names to target C++ class names.
	ServiceNameMapping map[string]string `yaml:"service_name_mapping,omitempty"`

	// ServiceNameToComment maps service names to comment overrides.
	ServiceNameToComment map[string]string `yaml:"service_name_to_comment,omitempty"`
}

// IdempotencyRule defines an idempotency override for an RPC.
type IdempotencyRule struct {
	// RPCName is the name of the RPC, optionally qualified by service name.
	RPCName string `yaml:"rpc_name"`

	// Idempotency is the idempotency classification (e.g., IDEMPOTENT, NON_IDEMPOTENT).
	Idempotency string `yaml:"idempotency"`
}
