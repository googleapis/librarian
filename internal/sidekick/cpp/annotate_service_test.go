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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService(t *testing.T) {
	for _, test := range []struct {
		name        string
		service     *api.Service
		sourceFile  string
		productPath string
		modelAnn    *modelAnnotations
		want        *serviceAnnotations
	}{
		{
			name:    "simple service without definition location",
			service: api.NewTestService("SimpleService"),
			modelAnn: &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			},
			want: &serviceAnnotations{
				Name:          "SimpleService",
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2026",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile:               "",
				InternalNamespace:        "internal",
				MocksNamespace:           "mocks",
				ClientHeaderPath:         "simple_client.h",
				ClientHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_SIMPLE_CLIENT_H",
				SourcesCcIncludes: []string{
					"internal/simple_auth_decorator.cc",
					"internal/simple_connection_impl.cc",
					"internal/simple_logging_decorator.cc",
					"internal/simple_metadata_decorator.cc",
					"internal/simple_option_defaults.cc",
					"internal/simple_stub.cc",
					"internal/simple_stub_factory.cc",
					"internal/simple_tracing_connection.cc",
					"internal/simple_tracing_stub.cc",
					"simple_client.cc",
					"simple_connection.cc",
					"simple_connection_idempotency_policy.cc",
				},
			},
		},
		{
			name:        "service with definition location and product path",
			service:     api.NewTestService("EchoService"),
			sourceFile:  "google/example/echo.proto",
			productPath: "google/cloud/echo/v1",
			modelAnn: &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			},
			want: &serviceAnnotations{
				Name:          "EchoService",
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2026",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile:               "google/example/echo.proto",
				Namespace:                "echo_v1",
				InternalNamespace:        "echo_v1_internal",
				MocksNamespace:           "echo_v1_mocks",
				ProtoHeaderPath:          "google/example/echo.pb.h",
				ProtoGrpcHeaderPath:      "google/example/echo.grpc.pb.h",
				ClientHeaderPath:         "google/cloud/echo/v1/echo_client.h",
				ClientHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_ECHO_CLIENT_H",
			},
		},
		{
			name:        "service with custom copyright year and golden path",
			service:     api.NewTestService("CustomYearService"),
			sourceFile:  "generator/integration_tests/test.proto",
			productPath: "generator/integration_tests/golden/v1",
			modelAnn: &modelAnnotations{
				CopyrightYear: "2024",
				BoilerPlate:   license.HeaderBulk(),
			},
			want: &serviceAnnotations{
				Name:          "CustomYearService",
				CopyrightYear: "2024",
				BoilerPlate:   license.HeaderBulk(),
				Model: &modelAnnotations{
					CopyrightYear: "2024",
					BoilerPlate:   license.HeaderBulk(),
				},
				SourceFile:               "generator/integration_tests/test.proto",
				Namespace:                "golden_v1",
				InternalNamespace:        "golden_v1_internal",
				MocksNamespace:           "golden_v1_mocks",
				ProtoHeaderPath:          "generator/integration_tests/test.pb.h",
				ProtoGrpcHeaderPath:      "generator/integration_tests/test.grpc.pb.h",
				ClientHeaderPath:         "generator/integration_tests/golden/v1/custom_year_client.h",
				ClientHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_CUSTOM_YEAR_CLIENT_H",
				ConnectionHeaderIncludes: []string{
					"generator/integration_tests/golden/v1/custom_year_connection_idempotency_policy.h",
					"generator/integration_tests/golden/v1/internal/custom_year_retry_traits.h",
				},
				ConnectionSourceIncludes: []string{
					"generator/integration_tests/golden/v1/custom_year_options.h",
					"generator/integration_tests/golden/v1/internal/custom_year_connection_impl.h",
					"generator/integration_tests/golden/v1/internal/custom_year_option_defaults.h",
					"generator/integration_tests/golden/v1/internal/custom_year_stub_factory.h",
					"generator/integration_tests/golden/v1/internal/custom_year_tracing_connection.h",
				},
				ConnectionImplHeaderIncludes: []string{
					"generator/integration_tests/golden/v1/custom_year_connection.h",
					"generator/integration_tests/golden/v1/custom_year_connection_idempotency_policy.h",
					"generator/integration_tests/golden/v1/custom_year_options.h",
					"generator/integration_tests/golden/v1/internal/custom_year_retry_traits.h",
					"generator/integration_tests/golden/v1/internal/custom_year_stub.h",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, nil, []*api.Service{test.service})
			if test.sourceFile != "" {
				model.DefinitionLocations = map[string]api.SourceLocation{
					test.service.ID: {Filename: test.sourceFile, Line: 42},
				}
			}
			libCfg := &config.CppLibrary{
				ProductPath: test.productPath,
			}
			c := newCodec(libCfg)
			if err := c.annotateService(test.service, test.modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got, ok := test.service.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("expected *serviceAnnotations, got %T", test.service.Codec)
			}
			ignoreFields := cmpopts.IgnoreFields(serviceAnnotations{},
				"ConnectionHeaderIncludeGuard",
				"IdempotencyPolicyHeaderIncludeGuard",
				"OptionsHeaderIncludeGuard",
				"MockConnectionHeaderIncludeGuard",
				"OptionDefaultsHeaderIncludeGuard",
				"RetryTraitsHeaderIncludeGuard",
				"TracingConnectionHeaderIncludeGuard",
				"ConnectionImplHeaderIncludeGuard",
				"StubFactoryHeaderIncludeGuard",
				"AuthDecoratorHeaderIncludeGuard",
				"LoggingDecoratorHeaderIncludeGuard",
				"MetadataDecoratorHeaderIncludeGuard",
				"StubHeaderIncludeGuard",
				"TracingStubHeaderIncludeGuard",
				"ForwardingClientHeaderIncludeGuard",
				"ForwardingConnectionHeaderIncludeGuard",
				"ForwardingIdempotencyPolicyHeaderIncludeGuard",
				"ForwardingOptionsHeaderIncludeGuard",
				"ForwardingMockConnectionHeaderIncludeGuard",
				"ConnectionHeaderPath",
				"IdempotencyPolicyHeaderPath",
				"OptionsHeaderPath",
				"MockConnectionHeaderPath",
				"OptionDefaultsHeaderPath",
				"RetryTraitsHeaderPath",
				"TracingConnectionHeaderPath",
				"ConnectionImplHeaderPath",
				"StubFactoryHeaderPath",
				"AuthDecoratorHeaderPath",
				"LoggingDecoratorHeaderPath",
				"MetadataDecoratorHeaderPath",
				"StubHeaderPath",
				"TracingStubHeaderPath",
				"ForwardingClientHeaderPath",
				"ForwardingConnectionHeaderPath",
				"ForwardingIdempotencyPolicyHeaderPath",
				"ForwardingOptionsHeaderPath",
				"ForwardingMockConnectionHeaderPath",
				"SourcesCcIncludes",
				"ConnectionHeaderIncludes",
				"ConnectionSourceIncludes",
				"ConnectionImplHeaderIncludes",
			)
			if test.want.SourcesCcIncludes != nil {
				if diff := cmp.Diff(test.want.SourcesCcIncludes, got.SourcesCcIncludes); diff != "" {
					t.Errorf("SourcesCcIncludes mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionHeaderIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionHeaderIncludes, got.ConnectionHeaderIncludes); diff != "" {
					t.Errorf("ConnectionHeaderIncludes mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionSourceIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionSourceIncludes, got.ConnectionSourceIncludes); diff != "" {
					t.Errorf("ConnectionSourceIncludes mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionImplHeaderIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionImplHeaderIncludes, got.ConnectionImplHeaderIncludes); diff != "" {
					t.Errorf("ConnectionImplHeaderIncludes mismatch (-want +got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(test.want, got, ignoreFields); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if test.service.Model != nil {
				t.Errorf("expected service.Model to not be mutated, got %v", test.service.Model)
			}
		})
	}
}

func TestAnnotateService_Gating(t *testing.T) {
	svc := api.NewTestService("GatedService")
	svc.Methods = []*api.Method{
		{
			Name:                "LroMethod",
			IsLRO:               true,
			OperationInfo:       &api.OperationInfo{},
			ClientSideStreaming: true,
			ServerSideStreaming: true,
			Pagination:          &api.Field{Name: "page_token"},
			AutoPopulated:       []*api.Field{{Name: "request_id"}},
			Routing:             []*api.RoutingInfo{{}},
		},
		{
			Name:                "ServerStreamingAsync",
			ServerSideStreaming: true,
		},
		{
			Name:                "ClientStreamingAsync",
			ClientSideStreaming: true,
		},
		{
			Name:                "ServerStreamingSync",
			ServerSideStreaming: true,
		},
		{
			Name:                "ClientStreamingSync",
			ClientSideStreaming: true,
		},
	}
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   license.HeaderBulk(),
	}
	libCfg := &config.CppLibrary{
		GenAsyncRPCs: []string{"ServerStreamingAsync", "GatedService.ClientStreamingAsync"},
	}
	c := newCodec(libCfg)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)

	if !got.HasLongrunningMethod {
		t.Errorf("expected HasLongrunningMethod to be true")
	}
	if !got.HasBidirStreamingMethod {
		t.Errorf("expected HasBidirStreamingMethod to be true")
	}
	if !got.HasPaginatedMethod {
		t.Errorf("expected HasPaginatedMethod to be true")
	}
	if !got.HasRequestId {
		t.Errorf("expected HasRequestId to be true")
	}
	if !got.HasExplicitRoutingMethod {
		t.Errorf("expected HasExplicitRoutingMethod to be true")
	}
	if !got.HasAsyncMethod {
		t.Errorf("expected HasAsyncMethod to be true (via LRO)")
	}
	if !got.HasStreamRange {
		t.Errorf("expected HasStreamRange to be true")
	}
	if !got.HasCompletionQueue {
		t.Errorf("expected HasCompletionQueue to be true")
	}
	if !got.HasAsyncStreamingReadMethod {
		t.Errorf("expected HasAsyncStreamingReadMethod to be true")
	}
	if !got.HasAsyncStreamingWriteMethod {
		t.Errorf("expected HasAsyncStreamingWriteMethod to be true")
	}
	if !got.HasStreamingReadMethod {
		t.Errorf("expected HasStreamingReadMethod to be true")
	}
	if !got.HasStreamingWriteMethod {
		t.Errorf("expected HasStreamingWriteMethod to be true")
	}
}

func TestAnnotateService_ConnectionIncludesSorting(t *testing.T) {
	svc := api.NewTestService("RequestIdService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2024",
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

	wantConnH := []string{
		"generator/integration_tests/golden/v1/internal/request_id_retry_traits.h",
		"generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.h",
	}
	if diff := cmp.Diff(wantConnH, got.ConnectionHeaderIncludes); diff != "" {
		t.Errorf("ConnectionHeaderIncludes mismatch (-want +got):\n%s", diff)
	}

	wantConnCC := []string{
		"generator/integration_tests/golden/v1/internal/request_id_connection_impl.h",
		"generator/integration_tests/golden/v1/internal/request_id_option_defaults.h",
		"generator/integration_tests/golden/v1/internal/request_id_stub_factory.h",
		"generator/integration_tests/golden/v1/internal/request_id_tracing_connection.h",
		"generator/integration_tests/golden/v1/request_id_options.h",
	}
	if diff := cmp.Diff(wantConnCC, got.ConnectionSourceIncludes); diff != "" {
		t.Errorf("ConnectionSourceIncludes mismatch (-want +got):\n%s", diff)
	}

	wantConnImplH := []string{
		"generator/integration_tests/golden/v1/internal/request_id_retry_traits.h",
		"generator/integration_tests/golden/v1/internal/request_id_stub.h",
		"generator/integration_tests/golden/v1/request_id_connection.h",
		"generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.h",
		"generator/integration_tests/golden/v1/request_id_options.h",
	}
	if diff := cmp.Diff(wantConnImplH, got.ConnectionImplHeaderIncludes); diff != "" {
		t.Errorf("ConnectionImplHeaderIncludes mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_Forwarding(t *testing.T) {
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

	wantClientFwdPath := "generator/integration_tests/golden/golden_kitchen_sink_client.h"
	if got.ForwardingClientHeaderPath != wantClientFwdPath {
		t.Errorf("ForwardingClientHeaderPath = %q, want %q", got.ForwardingClientHeaderPath, wantClientFwdPath)
	}
	wantFwdGuard := "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H"
	if got.ForwardingClientHeaderIncludeGuard != wantFwdGuard {
		t.Errorf("ForwardingClientHeaderIncludeGuard = %q, want %q", got.ForwardingClientHeaderIncludeGuard, wantFwdGuard)
	}
}
