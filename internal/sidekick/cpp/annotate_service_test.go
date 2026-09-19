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
				SourceFile:        "",
				InternalNamespace: "internal",
				MocksNamespace:    "mocks",

				ClientHeaderPath:            "simple_client.h",
				ConnectionHeaderPath:        "simple_connection.h",
				IdempotencyPolicyHeaderPath: "simple_connection_idempotency_policy.h",
				OptionsHeaderPath:           "simple_options.h",
				MockConnectionHeaderPath:    "mocks/mock_simple_connection.h",
				OptionDefaultsHeaderPath:    "internal/simple_option_defaults.h",
				RetryTraitsHeaderPath:       "internal/simple_retry_traits.h",
				TracingConnectionHeaderPath: "internal/simple_tracing_connection.h",
				ConnectionImplHeaderPath:    "internal/simple_connection_impl.h",
				StubFactoryHeaderPath:       "internal/simple_stub_factory.h",
				AuthDecoratorHeaderPath:     "internal/simple_auth_decorator.h",
				LoggingDecoratorHeaderPath:  "internal/simple_logging_decorator.h",
				MetadataDecoratorHeaderPath: "internal/simple_metadata_decorator.h",
				StubHeaderPath:              "internal/simple_stub.h",
				TracingStubHeaderPath:       "internal/simple_tracing_stub.h",

				ClientHeaderIncludeGuard:            "GOOGLE_CLOUD_CPP_SIMPLE_CLIENT_H",
				ConnectionHeaderIncludeGuard:        "GOOGLE_CLOUD_CPP_SIMPLE_CONNECTION_H",
				IdempotencyPolicyHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_SIMPLE_CONNECTION_IDEMPOTENCY_POLICY_H",
				OptionsHeaderIncludeGuard:           "GOOGLE_CLOUD_CPP_SIMPLE_OPTIONS_H",
				MockConnectionHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_MOCKS_MOCK_SIMPLE_CONNECTION_H",
				OptionDefaultsHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_OPTION_DEFAULTS_H",
				RetryTraitsHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_RETRY_TRAITS_H",
				TracingConnectionHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_TRACING_CONNECTION_H",
				ConnectionImplHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_CONNECTION_IMPL_H",
				StubFactoryHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_STUB_FACTORY_H",
				AuthDecoratorHeaderIncludeGuard:     "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_AUTH_DECORATOR_H",
				LoggingDecoratorHeaderIncludeGuard:  "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_LOGGING_DECORATOR_H",
				MetadataDecoratorHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_METADATA_DECORATOR_H",
				StubHeaderIncludeGuard:              "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_STUB_H",
				TracingStubHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_INTERNAL_SIMPLE_TRACING_STUB_H",

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
				ForwardingNamespace:      "",
				ForwardingMocksNamespace: "",
				ProtoHeaderPath:          "google/example/echo.pb.h",
				ProtoGrpcHeaderPath:      "google/example/echo.grpc.pb.h",

				ClientHeaderPath:            "google/cloud/echo/v1/echo_client.h",
				ConnectionHeaderPath:        "google/cloud/echo/v1/echo_connection.h",
				IdempotencyPolicyHeaderPath: "google/cloud/echo/v1/echo_connection_idempotency_policy.h",
				OptionsHeaderPath:           "google/cloud/echo/v1/echo_options.h",
				MockConnectionHeaderPath:    "google/cloud/echo/v1/mocks/mock_echo_connection.h",
				OptionDefaultsHeaderPath:    "google/cloud/echo/v1/internal/echo_option_defaults.h",
				RetryTraitsHeaderPath:       "google/cloud/echo/v1/internal/echo_retry_traits.h",
				TracingConnectionHeaderPath: "google/cloud/echo/v1/internal/echo_tracing_connection.h",
				ConnectionImplHeaderPath:    "google/cloud/echo/v1/internal/echo_connection_impl.h",
				StubFactoryHeaderPath:       "google/cloud/echo/v1/internal/echo_stub_factory.h",
				AuthDecoratorHeaderPath:     "google/cloud/echo/v1/internal/echo_auth_decorator.h",
				LoggingDecoratorHeaderPath:  "google/cloud/echo/v1/internal/echo_logging_decorator.h",
				MetadataDecoratorHeaderPath: "google/cloud/echo/v1/internal/echo_metadata_decorator.h",
				StubHeaderPath:              "google/cloud/echo/v1/internal/echo_stub.h",
				TracingStubHeaderPath:       "google/cloud/echo/v1/internal/echo_tracing_stub.h",

				ClientHeaderIncludeGuard:            "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_ECHO_CLIENT_H",
				ConnectionHeaderIncludeGuard:        "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_ECHO_CONNECTION_H",
				IdempotencyPolicyHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_ECHO_CONNECTION_IDEMPOTENCY_POLICY_H",
				OptionsHeaderIncludeGuard:           "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_ECHO_OPTIONS_H",
				MockConnectionHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_MOCKS_MOCK_ECHO_CONNECTION_H",
				OptionDefaultsHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_OPTION_DEFAULTS_H",
				RetryTraitsHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_RETRY_TRAITS_H",
				TracingConnectionHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_TRACING_CONNECTION_H",
				ConnectionImplHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_CONNECTION_IMPL_H",
				StubFactoryHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_STUB_FACTORY_H",
				AuthDecoratorHeaderIncludeGuard:     "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_AUTH_DECORATOR_H",
				LoggingDecoratorHeaderIncludeGuard:  "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_LOGGING_DECORATOR_H",
				MetadataDecoratorHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_METADATA_DECORATOR_H",
				StubHeaderIncludeGuard:              "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_STUB_H",
				TracingStubHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_ECHO_V1_INTERNAL_ECHO_TRACING_STUB_H",
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
				ForwardingNamespace:      "",
				ForwardingMocksNamespace: "",
				ProtoHeaderPath:          "generator/integration_tests/test.pb.h",
				ProtoGrpcHeaderPath:      "generator/integration_tests/test.grpc.pb.h",

				ClientHeaderPath:            "generator/integration_tests/golden/v1/custom_year_client.h",
				ConnectionHeaderPath:        "generator/integration_tests/golden/v1/custom_year_connection.h",
				IdempotencyPolicyHeaderPath: "generator/integration_tests/golden/v1/custom_year_connection_idempotency_policy.h",
				OptionsHeaderPath:           "generator/integration_tests/golden/v1/custom_year_options.h",
				MockConnectionHeaderPath:    "generator/integration_tests/golden/v1/mocks/mock_custom_year_connection.h",
				OptionDefaultsHeaderPath:    "generator/integration_tests/golden/v1/internal/custom_year_option_defaults.h",
				RetryTraitsHeaderPath:       "generator/integration_tests/golden/v1/internal/custom_year_retry_traits.h",
				TracingConnectionHeaderPath: "generator/integration_tests/golden/v1/internal/custom_year_tracing_connection.h",
				ConnectionImplHeaderPath:    "generator/integration_tests/golden/v1/internal/custom_year_connection_impl.h",
				StubFactoryHeaderPath:       "generator/integration_tests/golden/v1/internal/custom_year_stub_factory.h",
				AuthDecoratorHeaderPath:     "generator/integration_tests/golden/v1/internal/custom_year_auth_decorator.h",
				LoggingDecoratorHeaderPath:  "generator/integration_tests/golden/v1/internal/custom_year_logging_decorator.h",
				MetadataDecoratorHeaderPath: "generator/integration_tests/golden/v1/internal/custom_year_metadata_decorator.h",
				StubHeaderPath:              "generator/integration_tests/golden/v1/internal/custom_year_stub.h",
				TracingStubHeaderPath:       "generator/integration_tests/golden/v1/internal/custom_year_tracing_stub.h",

				ClientHeaderIncludeGuard:            "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_CUSTOM_YEAR_CLIENT_H",
				ConnectionHeaderIncludeGuard:        "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_CUSTOM_YEAR_CONNECTION_H",
				IdempotencyPolicyHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_CUSTOM_YEAR_CONNECTION_IDEMPOTENCY_POLICY_H",
				OptionsHeaderIncludeGuard:           "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_CUSTOM_YEAR_OPTIONS_H",
				MockConnectionHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_MOCKS_MOCK_CUSTOM_YEAR_CONNECTION_H",
				OptionDefaultsHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_OPTION_DEFAULTS_H",
				RetryTraitsHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_RETRY_TRAITS_H",
				TracingConnectionHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_TRACING_CONNECTION_H",
				ConnectionImplHeaderIncludeGuard:    "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_CONNECTION_IMPL_H",
				StubFactoryHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_STUB_FACTORY_H",
				AuthDecoratorHeaderIncludeGuard:     "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_AUTH_DECORATOR_H",
				LoggingDecoratorHeaderIncludeGuard:  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_LOGGING_DECORATOR_H",
				MetadataDecoratorHeaderIncludeGuard: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_METADATA_DECORATOR_H",
				StubHeaderIncludeGuard:              "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_STUB_H",
				TracingStubHeaderIncludeGuard:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_CUSTOM_YEAR_TRACING_STUB_H",
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
				"SourcesCcIncludes",
				"ConnectionHeaderIncludes",
				"ConnectionSourceIncludes",
				"ConnectionImplHeaderIncludes",
			)
			if test.want.SourcesCcIncludes != nil {
				if diff := cmp.Diff(test.want.SourcesCcIncludes, got.SourcesCcIncludes); diff != "" {
					t.Logf("SourcesCcIncludes")
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionHeaderIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionHeaderIncludes, got.ConnectionHeaderIncludes); diff != "" {
					t.Logf("ConnectionHeaderIncludes")
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionSourceIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionSourceIncludes, got.ConnectionSourceIncludes); diff != "" {
					t.Logf("ConnectionSourceIncludes")
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			if test.want.ConnectionImplHeaderIncludes != nil {
				if diff := cmp.Diff(test.want.ConnectionImplHeaderIncludes, got.ConnectionImplHeaderIncludes); diff != "" {
					t.Logf("ConnectionImplHeaderIncludes")
					t.Errorf("mismatch (-want +got):\n%s", diff)
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
	lroMethod := api.NewTestMethod("LroMethod").
		WithOperationInfo(&api.OperationInfo{}).
		WithBidiStreaming().
		WithPagination(api.NewTestField("page_token"))
	lroMethod.AutoPopulated = []*api.Field{api.NewTestField("request_id")}
	lroMethod.Routing = []*api.RoutingInfo{{}}

	svc := api.NewTestService("GatedService").WithMethods(
		lroMethod,
		api.NewTestMethod("ServerStreamingAsync").WithServerSideStreaming(),
		api.NewTestMethod("ClientStreamingAsync").WithClientSideStreaming(),
		api.NewTestMethod("ServerStreamingSync").WithServerSideStreaming(),
		api.NewTestMethod("ClientStreamingSync").WithClientSideStreaming(),
	)
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
		t.Logf("ConnectionHeaderIncludes")
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantConnCC := []string{
		"generator/integration_tests/golden/v1/internal/request_id_connection_impl.h",
		"generator/integration_tests/golden/v1/internal/request_id_option_defaults.h",
		"generator/integration_tests/golden/v1/internal/request_id_stub_factory.h",
		"generator/integration_tests/golden/v1/internal/request_id_tracing_connection.h",
		"generator/integration_tests/golden/v1/request_id_options.h",
	}
	if diff := cmp.Diff(wantConnCC, got.ConnectionSourceIncludes); diff != "" {
		t.Logf("ConnectionSourceIncludes")
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantConnImplH := []string{
		"generator/integration_tests/golden/v1/internal/request_id_retry_traits.h",
		"generator/integration_tests/golden/v1/internal/request_id_stub.h",
		"generator/integration_tests/golden/v1/request_id_connection.h",
		"generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.h",
		"generator/integration_tests/golden/v1/request_id_options.h",
	}
	if diff := cmp.Diff(wantConnImplH, got.ConnectionImplHeaderIncludes); diff != "" {
		t.Logf("ConnectionImplHeaderIncludes")
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

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

		for _, tc := range []struct {
			name string
			got  string
			want string
		}{
			{
				name: "ForwardingClientHeaderPath",
				got:  got.ForwardingClientHeaderPath,
				want: "generator/integration_tests/golden/golden_kitchen_sink_client.h",
			},
			{
				name: "ForwardingClientHeaderIncludeGuard",
				got:  got.ForwardingClientHeaderIncludeGuard,
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
			},
			{
				name: "ForwardingConnectionHeaderPath",
				got:  got.ForwardingConnectionHeaderPath,
				want: "generator/integration_tests/golden/golden_kitchen_sink_connection.h",
			},
			{
				name: "ForwardingConnectionHeaderIncludeGuard",
				got:  got.ForwardingConnectionHeaderIncludeGuard,
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_H",
			},
			{
				name: "ForwardingIdempotencyPolicyHeaderPath",
				got:  got.ForwardingIdempotencyPolicyHeaderPath,
				want: "generator/integration_tests/golden/golden_kitchen_sink_connection_idempotency_policy.h",
			},
			{
				name: "ForwardingIdempotencyPolicyHeaderIncludeGuard",
				got:  got.ForwardingIdempotencyPolicyHeaderIncludeGuard,
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CONNECTION_IDEMPOTENCY_POLICY_H",
			},
			{
				name: "ForwardingOptionsHeaderPath",
				got:  got.ForwardingOptionsHeaderPath,
				want: "generator/integration_tests/golden/golden_kitchen_sink_options.h",
			},
			{
				name: "ForwardingOptionsHeaderIncludeGuard",
				got:  got.ForwardingOptionsHeaderIncludeGuard,
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_OPTIONS_H",
			},
			{
				name: "ForwardingMockConnectionHeaderPath",
				got:  got.ForwardingMockConnectionHeaderPath,
				want: "generator/integration_tests/golden/mocks/mock_golden_kitchen_sink_connection.h",
			},
			{
				name: "ForwardingMockConnectionHeaderIncludeGuard",
				got:  got.ForwardingMockConnectionHeaderIncludeGuard,
				want: "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
			},
			{
				name: "Namespace",
				got:  got.Namespace,
				want: "golden_v1",
			},
			{
				name: "InternalNamespace",
				got:  got.InternalNamespace,
				want: "golden_v1_internal",
			},
			{
				name: "MocksNamespace",
				got:  got.MocksNamespace,
				want: "golden_v1_mocks",
			},
			{
				name: "ForwardingNamespace",
				got:  got.ForwardingNamespace,
				want: "golden",
			},
			{
				name: "ForwardingMocksNamespace",
				got:  got.ForwardingMocksNamespace,
				want: "golden_mocks",
			},
		} {
			if diff := cmp.Diff(tc.want, tc.got); diff != "" {
				t.Logf("%s", tc.name)
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
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

		if got.ForwardingNamespace != "" {
			t.Errorf("expected empty ForwardingNamespace, got %q", got.ForwardingNamespace)
		}
		if got.ForwardingMocksNamespace != "" {
			t.Errorf("expected empty ForwardingMocksNamespace, got %q", got.ForwardingMocksNamespace)
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

	if got.Namespace != "" {
		t.Errorf("expected empty Namespace for nil config, got %q", got.Namespace)
	}
	if got.InternalNamespace != "internal" {
		t.Errorf("expected 'internal' for nil config InternalNamespace, got %q", got.InternalNamespace)
	}
	if got.MocksNamespace != "mocks" {
		t.Errorf("expected 'mocks' for nil config MocksNamespace, got %q", got.MocksNamespace)
	}
	if got.ForwardingNamespace != "" {
		t.Errorf("expected empty ForwardingNamespace for nil config, got %q", got.ForwardingNamespace)
	}
	if got.ForwardingMocksNamespace != "" {
		t.Errorf("expected empty ForwardingMocksNamespace for nil config, got %q", got.ForwardingMocksNamespace)
	}
}
