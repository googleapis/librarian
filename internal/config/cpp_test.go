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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestCppConfig_Unmarshal(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want *Config
	}{
		{
			name: "parses cpp library with all 22 fields and tools",
			yaml: `
language: cpp
version: 1.0.0
tools:
  clang_format:
    path: /custom/bin/clang-format
    version: "18.0.0"
default:
  cpp:
    default_version: "2.3.0"
libraries:
  - name: testlib
    version: "2.3.0"
    cpp:
      source_root: google/cloud/test
      product_path: google/cloud/test/v1
      forwarding_product_path: google/cloud/test
      service_endpoint_env_var: TEST_ENDPOINT
      emulator_endpoint_env_var: TEST_EMULATOR
      generate_rest_transport: true
      generate_grpc_transport: true
      endpoint_location_style: LOCATION_DEPENDENT
      backwards_compatibility_namespace_alias: true
      omitted_rpcs:
        - DeprecatedRpc
      gen_async_rpcs:
        - LongRunningRpc
      omitted_services:
        - InternalService
      retryable_status_codes:
        - UNAVAILABLE
      idempotency_overrides:
        - rpc_name: TestService.CustomRpc
          idempotency: IDEMPOTENT
      generate_round_robin_decorator: true
      omit_client: false
      omit_connection: false
      omit_stub_factory: false
      additional_proto_files:
        - google/cloud/common.proto
      override_service_config_yaml_name: custom_service.yaml
      initial_copyright_year: "2020"
      omit_repo_metadata: true
`,
			want: func() *Config {
				grpcTransport := true
				return &Config{
					Language: LanguageCpp,
					Version:  "1.0.0",
					Tools: &Tools{
						ClangFormat: &ClangFormat{
							Path:    "/custom/bin/clang-format",
							Version: "18.0.0",
						},
					},
					Default: &Default{
						Cpp: &CppDefault{
							DefaultVersion: "2.3.0",
						},
					},
					Libraries: []*Library{
						{
							Name:    "testlib",
							Version: "2.3.0",
							Cpp: &CppLibrary{
								SourceRoot:                      "google/cloud/test",
								ProductPath:                     "google/cloud/test/v1",
								ForwardingProductPath:           "google/cloud/test",
								ServiceEndpointEnvVar:           "TEST_ENDPOINT",
								EmulatorEndpointEnvVar:          "TEST_EMULATOR",
								GenerateRestTransport:           true,
								GenerateGrpcTransport:           &grpcTransport,
								EndpointLocationStyle:           "LOCATION_DEPENDENT",
								BackwardsCompatibilityNamespace: true,
								OmittedRPCs:                     []string{"DeprecatedRpc"},
								GenAsyncRPCs:                    []string{"LongRunningRpc"},
								OmittedServices:                 []string{"InternalService"},
								RetryableStatusCodes:            []string{"UNAVAILABLE"},
								IdempotencyOverrides: []IdempotencyRule{
									{
										RPCName:     "TestService.CustomRpc",
										Idempotency: "IDEMPOTENT",
									},
								},
								GenerateRoundRobinDecorator:   true,
								OmitClient:                    false,
								OmitConnection:                false,
								OmitStubFactory:               false,
								AdditionalProtoFiles:          []string{"google/cloud/common.proto"},
								OverrideServiceConfigYAMLName: "custom_service.yaml",
								InitialCopyrightYear:          "2020",
								OmitRepoMetadata:              true,
							},
						},
					},
				}
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := yaml.Unmarshal[Config]([]byte(test.yaml))
			if err != nil {
				t.Fatalf("failed to unmarshal yaml: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
