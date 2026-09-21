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

package cpp

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestConvertConfig_Golden(t *testing.T) {
	data, err := os.ReadFile("testdata/golden_config.textproto")
	if err != nil {
		t.Fatal(err)
	}

	got, err := ConvertConfig(data, WithLibraryNameOverrides(map[string]string{
		"generator/integration_tests/test.proto":            "test_admin_database",
		"generator/integration_tests/test2.proto":           "test2_rest_only",
		"generator/integration_tests/test_request_id.proto": "test_request_id",
		"generator/integration_tests/test_deprecated.proto": "test_deprecated",
	}))
	if err != nil {
		t.Fatal(err)
	}

	grpcFalse := false
	grpcTrue := true

	want := &config.Config{
		Language: config.LanguageCpp,
		Libraries: []*config.Library{
			{
				Name: "test_admin_database",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					ServiceEndpointEnvVar:         "GOLDEN_KITCHEN_SINK_ENDPOINT",
					EmulatorEndpointEnvVar:        "GOLDEN_KITCHEN_SINK_EMULATOR_HOST",
					GenerateRestTransport:         true,
					ProductPath:                   "generator/integration_tests/golden/v1",
					ForwardingProductPath:         "generator/integration_tests/golden",
					OverrideServiceConfigYAMLName: "generator/integration_tests/test.yaml",
					InitialCopyrightYear:          "2022",
					GenerateRoundRobinDecorator:   true,
					OmitRepoMetadata:              true,
					AdditionalProtoFiles: []string{
						"generator/integration_tests/backup.proto",
					},
					OmittedRPCs: []string{
						"Omitted1",
						"GoldenKitchenSink.Omitted2",
						"Deprecated1",
						"Deprecated2(std::string const&)",
					},
					GenAsyncRPCs: []string{
						"GetDatabase",
						"DropDatabase",
						"StreamingRead",
						"StreamingWrite",
					},
					RetryableStatusCodes: []string{
						"GoldenKitchenSink.kInternal",
						"kUnavailable",
						"GoldenThingAdmin.kDeadlineExceeded",
					},
					IdempotencyOverrides: []config.IdempotencyRule{
						{RPCName: "GoldenThingAdmin.DropDatabase", Idempotency: "IDEMPOTENT"},
						{RPCName: "GoldenKitchenSink.ListLogs", Idempotency: "NON_IDEMPOTENT"},
					},
				},
			},
			{
				Name: "test2_rest_only",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					ProductPath:                   "generator/integration_tests/golden/v1",
					InitialCopyrightYear:          "2023",
					GenerateRestTransport:         true,
					GenerateGrpcTransport:         &grpcFalse,
					EndpointLocationStyle:         "LOCATION_OPTIONALLY_DEPENDENT",
					OverrideServiceConfigYAMLName: "generator/integration_tests/test2.yaml",
					RetryableStatusCodes: []string{
						"kUnavailable",
					},
				},
			},
			{
				Name: "test_request_id",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					ProductPath:                   "generator/integration_tests/golden/v1",
					InitialCopyrightYear:          "2024",
					OverrideServiceConfigYAMLName: "generator/integration_tests/test_request_id.yaml",
					GenAsyncRPCs: []string{
						"CreateFoo",
					},
					RetryableStatusCodes: []string{
						"kUnavailable",
					},
				},
			},
			{
				Name: "test_deprecated",
				APIs: []*config.API{
					{Path: "generator/integration_tests"},
				},
				Cpp: &config.CppLibrary{
					ProductPath:                   "generator/integration_tests/golden/v1",
					InitialCopyrightYear:          "2024",
					GenerateRestTransport:         true,
					GenerateGrpcTransport:         &grpcTrue,
					OverrideServiceConfigYAMLName: "generator/integration_tests/test_deprecated.yaml",
					RetryableStatusCodes: []string{
						"kUnavailable",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_Error(t *testing.T) {
	for _, test := range []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name: "unknown top-level field",
			input: `
unknown_top_level {
  field: "val"
}
`,
			errContains: `unknown top-level field "unknown_top_level" on line 2`,
		},
		{
			name: "unknown field in service",
			input: `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  invalid_service_field: "bad"
}
`,
			errContains: `unknown field "invalid_service_field" in service on line 4`,
		},
		{
			name: "unknown field in discovery_products",
			input: `
discovery_products {
  unknown_discovery_field: "bad"
}
`,
			errContains: `unknown field "unknown_discovery_field" in discovery_products on line 3`,
		},
		{
			name: "unknown field in idempotency_overrides",
			input: `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  idempotency_overrides: [
    {rpc_name: "Test.Rpc", bad_override: "xyz"}
  ]
}
`,
			errContains: `unknown field "bad_override" in idempotency_overrides on line 5`,
		},
		{
			name: "unknown field in service_name_mapping",
			input: `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  service_name_mapping {
    key: "Old"
    bad_val: "New"
  }
}
`,
			errContains: `unknown field "bad_val" in service_name_mapping on line 6`,
		},
		{
			name: "unknown field in service_name_to_comment",
			input: `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  service_name_to_comment {
    key: "Old"
    invalid_comment: "New"
  }
}
`,
			errContains: `unknown field "invalid_comment" in service_name_to_comment on line 6`,
		},
		{
			name: "unclosed string literal",
			input: `
service {
  service_proto_path: "unclosed string
}
`,
			errContains: `unclosed string literal on line 3`,
		},
		{
			name: "invalid boolean literal",
			input: `
service {
  service_proto_path: "google/cloud/test/v1/test.proto"
  generate_rest_transport: not_a_boolean
}
`,
			errContains: `expected boolean (true/false) on line 4, got "not_a_boolean"`,
		},
		{
			name: "service missing service_proto_path",
			input: `
service {
  product_path: "google/cloud/test/v1"
}
`,
			errContains: `service missing required service_proto_path on line 2`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConvertConfig([]byte(test.input))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", test.errContains)
			}
			if !strings.Contains(err.Error(), test.errContains) {
				t.Errorf("error %q does not contain expected substring %q", err.Error(), test.errContains)
			}
		})
	}
}

func TestConvertConfig_PubSubMappingAndComment(t *testing.T) {
	input := `
service {
  service_proto_path: "google/pubsub/v1/pubsub.proto"
  product_path: "google/cloud/pubsub/admin"
  initial_copyright_year: "2023"
  service_name_mapping { key: "Publisher" value: "TopicAdmin" }
  service_name_mapping { key: "Subscriber" value: "SubscriptionAdmin" }
  service_name_to_comment { key: "TopicAdmin" value: "A service to manipulate topics." }
  service_name_to_comment { key: "SubscriptionAdmin" value: "A service to manipulate subscriptions." }
}
`
	got, err := ConvertConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(got.Libraries))
	}
	lib := got.Libraries[0]
	if diff := cmp.Diff("pubsub_admin", lib.Name); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantMapping := map[string]string{
		"Publisher":  "TopicAdmin",
		"Subscriber": "SubscriptionAdmin",
	}
	if diff := cmp.Diff(wantMapping, lib.Cpp.ServiceNameMapping); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantComment := map[string]string{
		"TopicAdmin":        "A service to manipulate topics.",
		"SubscriptionAdmin": "A service to manipulate subscriptions.",
	}
	if diff := cmp.Diff(wantComment, lib.Cpp.ServiceNameToComment); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfig_DiscoveryProducts(t *testing.T) {
	input := `
discovery_products {
  discovery_document_url: "file:///workspace/generator/discovery/compute_public_google_rest_v1.json"
  operation_services: ["GlobalOperations", "RegionOperations"]

  rest_services {
    service_proto_path: "google/cloud/compute/accelerator_types/v1/accelerator_types.proto"
    product_path: "google/cloud/compute/accelerator_types/v1"
    initial_copyright_year: "2023"
    generate_rest_transport: true
    generate_grpc_transport: false
  }

  rest_services {
    service_proto_path: "google/cloud/compute/addresses/v1/addresses.proto"
    product_path: "google/cloud/compute/addresses/v1"
    initial_copyright_year: "2023"
    generate_rest_transport: true
    generate_grpc_transport: false
  }
}
`
	got, err := ConvertConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Libraries) != 2 {
		t.Fatalf("expected 2 libraries, got %d", len(got.Libraries))
	}
	if diff := cmp.Diff("compute_accelerator_types", got.Libraries[0].Name); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("compute_addresses", got.Libraries[1].Name); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestConvertConfigToYAML(t *testing.T) {
	input := `
service {
  service_proto_path: "google/cloud/secretmanager/v1/service.proto"
  product_path: "google/cloud/secretmanager/v1"
  forwarding_product_path: "google/cloud/secretmanager"
  initial_copyright_year: "2024"
  gen_async_rpcs: ["CreateFoo"]
}
`
	yamlBytes, err := ConvertConfigToYAML([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	unmarshaled, err := yaml.Unmarshal[config.Config](yamlBytes)
	if err != nil {
		t.Fatal(err)
	}

	if len(unmarshaled.Libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(unmarshaled.Libraries))
	}
	if diff := cmp.Diff("secretmanager", unmarshaled.Libraries[0].Name); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("2024", unmarshaled.Libraries[0].Cpp.InitialCopyrightYear); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDeriveLibraryName(t *testing.T) {
	for _, test := range []struct {
		name                  string
		serviceProtoPath      string
		productPath           string
		forwardingProductPath string
		want                  string
	}{
		{
			name:             "golden integration tests test.proto",
			serviceProtoPath: "generator/integration_tests/test.proto",
			want:             "test",
		},
		{
			name:             "golden integration tests request_id",
			serviceProtoPath: "generator/integration_tests/test_request_id.proto",
			want:             "test_request_id",
		},
		{
			name:                  "forwarding product path",
			serviceProtoPath:      "google/cloud/accessapproval/v1/accessapproval.proto",
			productPath:           "google/cloud/accessapproval/v1",
			forwardingProductPath: "google/cloud/accessapproval",
			want:                  "accessapproval",
		},
		{
			name:                  "secretmanager with service.proto",
			serviceProtoPath:      "google/cloud/secretmanager/v1/service.proto",
			productPath:           "google/cloud/secretmanager/v1",
			forwardingProductPath: "google/cloud/secretmanager",
			want:                  "secretmanager",
		},
		{
			name:                  "forwarding product path disambiguation",
			serviceProtoPath:      "google/cloud/kms/v1/ekm_service.proto",
			productPath:           "google/cloud/kms/v1",
			forwardingProductPath: "google/cloud/kms",
			want:                  "kms_ekm_service",
		},
		{
			name:             "aiplatform dataset service without forwarding path",
			serviceProtoPath: "google/cloud/aiplatform/v1/dataset_service.proto",
			productPath:      "google/cloud/aiplatform/v1",
			want:             "aiplatform_dataset_service",
		},
		{
			name:             "fallback to base proto filename",
			serviceProtoPath: "custom/path/my_service.proto",
			want:             "my_service",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveLibraryName(test.serviceProtoPath, test.productPath, test.forwardingProductPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
