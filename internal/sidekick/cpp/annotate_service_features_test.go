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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_Forwarding(t *testing.T) {
	t.Run("configured forwarding", func(t *testing.T) {
		svc := api.NewTestService("GoldenKitchenSink")
		model := api.NewTestAPI(nil, nil, []*api.Service{svc})
		modelAnn := &modelAnnotations{
			CopyrightYear: "2026",
			BoilerPlate:   license.HeaderBulk(),
		}
		libCfg := &config.CppLibrary{
			ProductPath:           "generator/integration_tests/golden/v1",
			ForwardingProductPath: "generator/integration_tests/golden",
		}
		c := newCodec(libCfg)
		if err := c.annotateService(svc, modelAnn, model); err != nil {
			t.Fatal(err)
		}
		got := svc.Codec.(*serviceAnnotations)

		for _, test := range []struct {
			name string
			want string
			got  string
		}{
			{
				name: "ForwardingClientHeaderPath",
				want: "generator/integration_tests/golden/golden_kitchen_sink_client.h",
				got:  got.ForwardingClientHeaderPath,
			},
			{
				name: "ForwardingClientHeaderIncludeGuard",
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
				got:  got.ForwardingClientHeaderIncludeGuard,
			},
			{
				name: "ForwardingConnectionHeaderPath",
				want: "generator/integration_tests/golden/golden_kitchen_sink_connection.h",
				got:  got.ForwardingConnectionHeaderPath,
			},
			{
				name: "ForwardingConnectionHeaderIncludeGuard",
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_H",
				got:  got.ForwardingConnectionHeaderIncludeGuard,
			},
			{
				name: "ForwardingIdempotencyPolicyHeaderPath",
				want: "generator/integration_tests/golden/golden_kitchen_sink_connection_idempotency_policy.h",
				got:  got.ForwardingIdempotencyPolicyHeaderPath,
			},
			{
				name: "ForwardingIdempotencyPolicyHeaderIncludeGuard",
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_IDEMPOTENCY_POLICY_H",
				got:  got.ForwardingIdempotencyPolicyHeaderIncludeGuard,
			},
			{
				name: "ForwardingOptionsHeaderPath",
				want: "generator/integration_tests/golden/golden_kitchen_sink_options.h",
				got:  got.ForwardingOptionsHeaderPath,
			},
			{
				name: "ForwardingOptionsHeaderIncludeGuard",
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_OPTIONS_H",
				got:  got.ForwardingOptionsHeaderIncludeGuard,
			},
			{
				name: "ForwardingMockConnectionHeaderPath",
				want: "generator/integration_tests/golden/mocks/mock_golden_kitchen_sink_connection.h",
				got:  got.ForwardingMockConnectionHeaderPath,
			},
			{
				name: "ForwardingMockConnectionHeaderIncludeGuard",
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
				got:  got.ForwardingMockConnectionHeaderIncludeGuard,
			},
			{
				name: "Namespace",
				want: "golden_v1",
				got:  got.Namespace,
			},
			{
				name: "InternalNamespace",
				want: "golden_v1_internal",
				got:  got.InternalNamespace,
			},
			{
				name: "MocksNamespace",
				want: "golden_v1_mocks",
				got:  got.MocksNamespace,
			},
			{
				name: "ForwardingNamespace",
				want: "golden",
				got:  got.ForwardingNamespace,
			},
			{
				name: "ForwardingMocksNamespace",
				want: "golden_mocks",
				got:  got.ForwardingMocksNamespace,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				if diff := cmp.Diff(test.want, test.got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("empty forwarding product path", func(t *testing.T) {
		svc := api.NewTestService("GoldenKitchenSink")
		model := api.NewTestAPI(nil, nil, []*api.Service{svc})
		modelAnn := &modelAnnotations{
			CopyrightYear: "2026",
			BoilerPlate:   license.HeaderBulk(),
		}
		libCfg := &config.CppLibrary{
			ProductPath: "generator/integration_tests/golden/v1",
		}
		c := newCodec(libCfg)
		if err := c.annotateService(svc, modelAnn, model); err != nil {
			t.Fatal(err)
		}
		got := svc.Codec.(*serviceAnnotations)

		for _, test := range []struct {
			name string
			want string
			got  string
		}{
			{name: "ForwardingNamespace", want: "", got: got.ForwardingNamespace},
			{name: "ForwardingMocksNamespace", want: "", got: got.ForwardingMocksNamespace},
		} {
			t.Run(test.name, func(t *testing.T) {
				if diff := cmp.Diff(test.want, test.got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}

func TestAnnotateService_NilConfig(t *testing.T) {
	svc := api.NewTestService("SimpleService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   license.HeaderBulk(),
	}
	c := newCodec(nil)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)

	for _, test := range []struct {
		name string
		want string
		got  string
	}{
		{name: "Namespace", want: "", got: got.Namespace},
		{name: "InternalNamespace", want: "internal", got: got.InternalNamespace},
		{name: "MocksNamespace", want: "mocks", got: got.MocksNamespace},
		{name: "ForwardingNamespace", want: "", got: got.ForwardingNamespace},
		{name: "ForwardingMocksNamespace", want: "", got: got.ForwardingMocksNamespace},
	} {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.want, test.got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_RetryStatusCodes(t *testing.T) {
	for _, test := range []struct {
		name                 string
		serviceName          string
		retryableStatusCodes []string
		want                 []string
	}{
		{
			name:                 "default codes when unset",
			serviceName:          "TestService",
			retryableStatusCodes: nil,
			want:                 []string{"kDeadlineExceeded", "kUnavailable"},
		},
		{
			name:                 "global codes only",
			serviceName:          "TestService",
			retryableStatusCodes: []string{"kUnavailable"},
			want:                 []string{"kUnavailable"},
		},
		{
			name:                 "service-scoped codes matching service",
			serviceName:          "TestService",
			retryableStatusCodes: []string{"TestService.kAborted", "kUnavailable", "OtherService.kNotFound"},
			want:                 []string{"kAborted", "kUnavailable"},
		},
		{
			name:                 "service-scoped codes not matching falls back to global",
			serviceName:          "TestService",
			retryableStatusCodes: []string{"OtherService.kNotFound", "kUnavailable"},
			want:                 []string{"kUnavailable"},
		},
		{
			name:                 "deduplicated and sorted",
			serviceName:          "TestService",
			retryableStatusCodes: []string{"TestService.kUnavailable", "kUnavailable", "kAborted"},
			want:                 []string{"kAborted", "kUnavailable"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService(test.serviceName)
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			libCfg := &config.CppLibrary{
				RetryableStatusCodes: test.retryableStatusCodes,
			}
			c := newCodec(libCfg)
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			if diff := cmp.Diff(test.want, got.RetryStatusCodes); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_DescriptionLinesAndProductOptionsPage(t *testing.T) {
	for _, test := range []struct {
		name                   string
		serviceName            string
		documentation          string
		productPath            string
		wantDescriptionLines   []docLine
		wantProductOptionsPage string
	}{
		{
			name:          "default documentation with golden product path",
			serviceName:   "RequestIdService",
			documentation: "",
			productPath:   "generator/integration_tests/golden/v1",
			wantDescriptionLines: []docLine{
				{Text: "RequestIdServiceClient", HasContent: true},
			},
			wantProductOptionsPage: "generator-integration_tests-golden-options",
		},
		{
			name:          "custom documentation with multiple lines",
			serviceName:   "EchoService",
			documentation: "Service for echo.\n\nAdditional details.",
			productPath:   "google/cloud/echo/v1",
			wantDescriptionLines: []docLine{
				{Text: "Service for echo.", HasContent: true},
				{Text: "", HasContent: false},
				{Text: "Additional details.", HasContent: true},
			},
			wantProductOptionsPage: "google-cloud-echo-options",
		},
		{
			name:                   "empty product path",
			serviceName:            "SimpleService",
			documentation:          "",
			productPath:            "",
			wantProductOptionsPage: "options",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService(test.serviceName)
			svc.Documentation = test.documentation
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			libCfg := &config.CppLibrary{
				ProductPath: test.productPath,
			}
			c := newCodec(libCfg)
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			if test.wantDescriptionLines != nil {
				if diff := cmp.Diff(test.wantDescriptionLines, got.DescriptionLines); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(test.wantProductOptionsPage, got.ProductOptionsPage); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_ComputeLRO(t *testing.T) {
	type computeLroSummary struct {
		HasLongrunningMethod                  bool
		HasGrpcLRO                            bool
		HasComputeLRO                         bool
		HasRegionOperations                   bool
		HasGlobalOperations                   bool
		HasGlobalOrganizationOperations       bool
		HasZoneOperations                     bool
		LongrunningOperationType              string
		LongrunningGetOperationRequestType    string
		LongrunningCancelOperationRequestType string
		LongrunningOperationIncludeHeader     string
	}
	for _, test := range []struct {
		name      string
		opService string
		want      computeLroSummary
	}{
		{
			name:      "RegionOperations",
			opService: "RegionOperations",
			want: computeLroSummary{
				HasLongrunningMethod:                  true,
				HasGrpcLRO:                            false,
				HasComputeLRO:                         true,
				HasRegionOperations:                   true,
				LongrunningOperationType:              "google::cloud::cpp::compute::v1::Operation",
				LongrunningGetOperationRequestType:    "google::cloud::cpp::compute::region_operations::v1::GetOperationRequest",
				LongrunningCancelOperationRequestType: "google::cloud::cpp::compute::region_operations::v1::DeleteOperationRequest",
				LongrunningOperationIncludeHeader:     "google/cloud/compute/region_operations/v1/region_operations.pb.h",
			},
		},
		{
			name:      "GlobalOperations",
			opService: "GlobalOperations",
			want: computeLroSummary{
				HasLongrunningMethod:                  true,
				HasGrpcLRO:                            false,
				HasComputeLRO:                         true,
				HasGlobalOperations:                   true,
				LongrunningOperationType:              "google::cloud::cpp::compute::v1::Operation",
				LongrunningGetOperationRequestType:    "google::cloud::cpp::compute::global_operations::v1::GetOperationRequest",
				LongrunningCancelOperationRequestType: "google::cloud::cpp::compute::global_operations::v1::DeleteOperationRequest",
				LongrunningOperationIncludeHeader:     "google/cloud/compute/global_operations/v1/global_operations.pb.h",
			},
		},
		{
			name:      "GlobalOrganizationOperations",
			opService: "GlobalOrganizationOperations",
			want: computeLroSummary{
				HasLongrunningMethod:                  true,
				HasGrpcLRO:                            false,
				HasComputeLRO:                         true,
				HasGlobalOrganizationOperations:       true,
				LongrunningOperationType:              "google::cloud::cpp::compute::v1::Operation",
				LongrunningGetOperationRequestType:    "google::cloud::cpp::compute::global_organization_operations::v1::GetOperationRequest",
				LongrunningCancelOperationRequestType: "google::cloud::cpp::compute::global_organization_operations::v1::DeleteOperationRequest",
				LongrunningOperationIncludeHeader:     "google/cloud/compute/global_organization_operations/v1/global_organization_operations.pb.h",
			},
		},
		{
			name:      "ZoneOperations",
			opService: "ZoneOperations",
			want: computeLroSummary{
				HasLongrunningMethod:                  true,
				HasGrpcLRO:                            false,
				HasComputeLRO:                         true,
				HasZoneOperations:                     true,
				LongrunningOperationType:              "google::cloud::cpp::compute::v1::Operation",
				LongrunningGetOperationRequestType:    "google::cloud::cpp::compute::zone_operations::v1::GetOperationRequest",
				LongrunningCancelOperationRequestType: "google::cloud::cpp::compute::zone_operations::v1::DeleteOperationRequest",
				LongrunningOperationIncludeHeader:     "google/cloud/compute/zone_operations/v1/zone_operations.pb.h",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			op := api.NewTestMessage("Operation").WithPackage("google.cloud.cpp.compute.v1")
			req := api.NewTestMessage("InsertRequest")
			method := api.NewTestMethod("Insert").
				WithInput(req).
				WithOutput(op).
				WithOperationService(test.opService)
			svc := api.NewTestService("ComputeService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc})
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			libCfg := &config.CppLibrary{GenerateRestTransport: true}
			c := newCodec(libCfg)
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			gotSummary := computeLroSummary{
				HasLongrunningMethod:                  got.HasLongrunningMethod,
				HasGrpcLRO:                            got.HasGrpcLRO,
				HasComputeLRO:                         got.HasComputeLRO,
				HasRegionOperations:                   got.HasRegionOperations,
				HasGlobalOperations:                   got.HasGlobalOperations,
				HasGlobalOrganizationOperations:       got.HasGlobalOrganizationOperations,
				HasZoneOperations:                     got.HasZoneOperations,
				LongrunningOperationType:              got.LongrunningOperationType,
				LongrunningGetOperationRequestType:    got.LongrunningGetOperationRequestType,
				LongrunningCancelOperationRequestType: got.LongrunningCancelOperationRequestType,
				LongrunningOperationIncludeHeader:     got.LongrunningOperationIncludeHeader,
			}
			if diff := cmp.Diff(test.want, gotSummary); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_PaginationFallback_HasPaginatedMethodAndStreamRange(t *testing.T) {
	// Method where Pagination is not pre-populated on api.Method, but conforms to pagination schema
	req := api.NewTestMessage("ListFoosRequest").WithFields(
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("page_token").WithType(api.TypezString),
	)
	resp := api.NewTestMessage("ListFoosResponse").WithFields(
		api.NewTestField("foos").WithType(api.TypezString).WithRepeated(),
		api.NewTestField("next_page_token").WithType(api.TypezString),
	)
	method := api.NewTestMethod("ListFoos").
		WithInput(req).
		WithOutput(resp)
	svc := api.NewTestService("FooService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   license.HeaderBulk(),
	}
	c := newCodec(nil)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	type paginationServiceSummary struct {
		HasPaginatedMethod bool
		HasStreamRange     bool
	}
	want := paginationServiceSummary{
		HasPaginatedMethod: true,
		HasStreamRange:     true,
	}
	gotSummary := paginationServiceSummary{
		HasPaginatedMethod: got.HasPaginatedMethod,
		HasStreamRange:     got.HasStreamRange,
	}
	if diff := cmp.Diff(want, gotSummary); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_StandardAIP151_RESTTransportOnly_UsesStandardOperation(t *testing.T) {
	op := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	req := api.NewTestMessage("InsertRequest")
	method := api.NewTestMethod("Insert").
		WithInput(req).
		WithOutput(op)
	method.IsLRO = true
	svc := api.NewTestService("StandardLroService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   license.HeaderBulk(),
	}
	generateGrpcTransport := false
	libCfg := &config.CppLibrary{
		GenerateRestTransport: true,
		GenerateGrpcTransport: &generateGrpcTransport,
	}
	c := newCodec(libCfg)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	type lroServiceSummary struct {
		HasLongrunningMethod     bool
		HasGrpcLRO               bool
		HasComputeLRO            bool
		LongrunningOperationType string
	}
	want := lroServiceSummary{
		HasLongrunningMethod:     true,
		HasGrpcLRO:               false,
		HasComputeLRO:            false,
		LongrunningOperationType: "google::longrunning::Operation",
	}
	gotSummary := lroServiceSummary{
		HasLongrunningMethod:     got.HasLongrunningMethod,
		HasGrpcLRO:               got.HasGrpcLRO,
		HasComputeLRO:            got.HasComputeLRO,
		LongrunningOperationType: got.LongrunningOperationType,
	}
	if diff := cmp.Diff(want, gotSummary); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
