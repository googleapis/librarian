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
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_SourcesCcCopyrightYear(t *testing.T) {
	for _, test := range []struct {
		name      string
		inputYear string
		wantYear  string
	}{
		{name: "year prior to 2024 clamped to 2024", inputYear: "2022", wantYear: "2024"},
		{name: "year 2024 preserved", inputYear: "2024", wantYear: "2024"},
		{name: "year 2026 preserved", inputYear: "2026", wantYear: "2026"},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("EchoService")
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			c := newCodec(&config.CppLibrary{InitialCopyrightYear: test.inputYear})
			modelAnn := &modelAnnotations{
				CopyrightYear: test.inputYear,
				BoilerPlate:   []string{"// Sample Boilerplate"},
			}
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			if diff := cmp.Diff(test.wantYear, got.SourcesCcCopyrightYear); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_StubProtoIncludes_ThreePhaseSort(t *testing.T) {
	lroMethod := api.NewTestMethod("LongRunning").
		WithOperationInfo(&api.OperationInfo{})
	svc := api.NewTestService("EchoService").
		WithMethods(lroMethod)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	model.DefinitionLocations = map[string]api.SourceLocation{
		svc.ID: {Filename: "google/example/echo.proto", Line: 1},
	}

	libCfg := &config.CppLibrary{
		ProductPath:          "google/example/v1",
		AdditionalProtoFiles: []string{"google/example/z_extra.proto", "google/example/a_extra.proto"},
	}
	c := newCodec(libCfg)
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   []string{"// Sample Boilerplate"},
	}
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)

	wantStubIncludes := []string{
		"google/example/a_extra.pb.h",
		"google/example/z_extra.pb.h",
		"google/example/echo.grpc.pb.h",
		"google/longrunning/operations.grpc.pb.h",
	}
	if diff := cmp.Diff(wantStubIncludes, got.StubProtoIncludes); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantConnIncludes := []string{
		"google/example/a_extra.pb.h",
		"google/example/echo.pb.h",
		"google/example/z_extra.pb.h",
	}
	if diff := cmp.Diff(wantConnIncludes, got.ConnectionProtoIncludes); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantIdempotencyIncludes := []string{
		"google/example/echo.grpc.pb.h",
	}
	if diff := cmp.Diff(wantIdempotencyIncludes, got.IdempotencyPolicyProtoIncludes); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_RestStubProtoIncludes(t *testing.T) {
	method := api.NewTestMethod("Unary").
		WithInput(api.NewTestMessage("Request")).
		WithOutput(api.NewTestMessage("Response")).
		WithVerb("POST").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("echo"))
	svc := api.NewTestService("EchoService").
		WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	model.DefinitionLocations = map[string]api.SourceLocation{
		svc.ID: {Filename: "google/example/echo.proto", Line: 1},
	}
	libCfg := &config.CppLibrary{
		ProductPath:           "google/example/v1",
		AdditionalProtoFiles:  []string{"google/example/extra.proto"},
		GenerateRestTransport: true,
		GenAsyncRPCs:          []string{"Unary"},
	}
	c := newCodec(libCfg)
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   []string{"// Sample Boilerplate"},
	}
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	if len(got.RestMethods) != 1 || got.RestMethods[0].Name != "Unary" {
		t.Errorf("expected 1 RestMethod 'Unary', got %v", got.RestMethods)
	}
	if len(got.RestAsyncMethods) != 1 || got.RestAsyncMethods[0].Name != "Unary" {
		t.Errorf("expected 1 RestAsyncMethod 'Unary', got %v", got.RestAsyncMethods)
	}
	wantIncludes := []string{
		"google/example/extra.pb.h",
		"google/example/echo.pb.h",
	}
	if diff := cmp.Diff(wantIncludes, got.RestStubProtoIncludes); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestHelper_ConnectionHeaderIncludes(t *testing.T) {
	got := connectionHeaderIncludes("b_path.h", "a_path.h")
	want := []string{"a_path.h", "b_path.h"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestHelper_ConnectionSourceIncludes(t *testing.T) {
	for _, test := range []struct {
		name                  string
		generateGrpcTransport bool
		want                  []string
	}{
		{
			name:                  "with gRPC",
			generateGrpcTransport: true,
			want:                  []string{"conn_impl.h", "opt_def.h", "options.h", "stub_factory.h", "tracing.h"},
		},
		{
			name:                  "without gRPC",
			generateGrpcTransport: false,
			want:                  []string{"opt_def.h", "options.h", "tracing.h"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := connectionSourceIncludes(
				"opt_def.h",
				"tracing.h",
				"options.h",
				"conn_impl.h",
				"stub_factory.h",
				test.generateGrpcTransport,
			)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_SourcesCcIncludes_Flags(t *testing.T) {
	for _, test := range []struct {
		name                        string
		generateRestTransport       bool
		generateGrpcTransport       bool
		generateRoundRobinDecorator bool
		want                        []string
	}{
		{
			name: "all false",
			want: []string{
				"google/cloud/test/echo_client.cc",
				"google/cloud/test/echo_connection.cc",
				"google/cloud/test/echo_connection_idempotency_policy.cc",
				"google/cloud/test/internal/echo_option_defaults.cc",
				"google/cloud/test/internal/echo_tracing_connection.cc",
			},
		},
		{
			name:                        "with round robin",
			generateRoundRobinDecorator: true,
			want: []string{
				"google/cloud/test/echo_client.cc",
				"google/cloud/test/echo_connection.cc",
				"google/cloud/test/echo_connection_idempotency_policy.cc",
				"google/cloud/test/internal/echo_option_defaults.cc",
				"google/cloud/test/internal/echo_round_robin_decorator.cc",
				"google/cloud/test/internal/echo_tracing_connection.cc",
			},
		},
		{
			name:                  "with rest transport",
			generateRestTransport: true,
			want: []string{
				"google/cloud/test/echo_client.cc",
				"google/cloud/test/echo_connection.cc",
				"google/cloud/test/echo_connection_idempotency_policy.cc",
				"google/cloud/test/echo_rest_connection.cc",
				"google/cloud/test/internal/echo_option_defaults.cc",
				"google/cloud/test/internal/echo_rest_connection_impl.cc",
				"google/cloud/test/internal/echo_rest_logging_decorator.cc",
				"google/cloud/test/internal/echo_rest_metadata_decorator.cc",
				"google/cloud/test/internal/echo_rest_stub.cc",
				"google/cloud/test/internal/echo_rest_stub_factory.cc",
				"google/cloud/test/internal/echo_tracing_connection.cc",
			},
		},
		{
			name:                  "with grpc transport",
			generateGrpcTransport: true,
			want: []string{
				"google/cloud/test/echo_client.cc",
				"google/cloud/test/echo_connection.cc",
				"google/cloud/test/echo_connection_idempotency_policy.cc",
				"google/cloud/test/internal/echo_auth_decorator.cc",
				"google/cloud/test/internal/echo_connection_impl.cc",
				"google/cloud/test/internal/echo_logging_decorator.cc",
				"google/cloud/test/internal/echo_metadata_decorator.cc",
				"google/cloud/test/internal/echo_option_defaults.cc",
				"google/cloud/test/internal/echo_stub.cc",
				"google/cloud/test/internal/echo_stub_factory.cc",
				"google/cloud/test/internal/echo_tracing_connection.cc",
				"google/cloud/test/internal/echo_tracing_stub.cc",
			},
		},
		{
			name:                        "all flags true",
			generateRestTransport:       true,
			generateGrpcTransport:       true,
			generateRoundRobinDecorator: true,
			want: []string{
				"google/cloud/test/echo_client.cc",
				"google/cloud/test/echo_connection.cc",
				"google/cloud/test/echo_connection_idempotency_policy.cc",
				"google/cloud/test/echo_rest_connection.cc",
				"google/cloud/test/internal/echo_auth_decorator.cc",
				"google/cloud/test/internal/echo_connection_impl.cc",
				"google/cloud/test/internal/echo_logging_decorator.cc",
				"google/cloud/test/internal/echo_metadata_decorator.cc",
				"google/cloud/test/internal/echo_option_defaults.cc",
				"google/cloud/test/internal/echo_rest_connection_impl.cc",
				"google/cloud/test/internal/echo_rest_logging_decorator.cc",
				"google/cloud/test/internal/echo_rest_metadata_decorator.cc",
				"google/cloud/test/internal/echo_rest_stub.cc",
				"google/cloud/test/internal/echo_rest_stub_factory.cc",
				"google/cloud/test/internal/echo_round_robin_decorator.cc",
				"google/cloud/test/internal/echo_stub.cc",
				"google/cloud/test/internal/echo_stub_factory.cc",
				"google/cloud/test/internal/echo_tracing_connection.cc",
				"google/cloud/test/internal/echo_tracing_stub.cc",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sourcesCcIncludes(
				"google/cloud/test",
				"Echo",
				test.generateRestTransport,
				test.generateGrpcTransport,
				test.generateRoundRobinDecorator,
			)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_ConnectionProtoIncludes_EdgeCases(t *testing.T) {
	for _, test := range []struct {
		name                 string
		protoHeaderPath      string
		additionalProtoFiles []string
		want                 []string
	}{
		{
			name:                 "empty inputs",
			protoHeaderPath:      "",
			additionalProtoFiles: nil,
			want:                 nil,
		},
		{
			name:                 "deduplication and sort",
			protoHeaderPath:      "b.pb.h",
			additionalProtoFiles: []string{"b.proto", "a.proto", "a.proto"},
			want:                 []string{"a.pb.h", "b.pb.h"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := connectionProtoIncludes(test.protoHeaderPath, test.additionalProtoFiles)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_IdempotencyPolicyProtoIncludes_EdgeCases(t *testing.T) {
	for _, test := range []struct {
		name                  string
		generateGrpcTransport bool
		protoGrpcHeaderPath   string
		protoHeaderPath       string
		hasLocationMixin      bool
		hasIamMixin           bool
		hasOperationsMixin    bool
		want                  []string
	}{
		{
			name:                  "rest only with empty proto header",
			generateGrpcTransport: false,
			protoGrpcHeaderPath:   "grpc.pb.h",
			protoHeaderPath:       "",
			hasLocationMixin:      false,
			hasIamMixin:           false,
			hasOperationsMixin:    false,
			want:                  nil,
		},
		{
			name:                  "rest only with proto header ignores mixins",
			generateGrpcTransport: false,
			protoGrpcHeaderPath:   "grpc.pb.h",
			protoHeaderPath:       "proto.pb.h",
			hasLocationMixin:      true,
			hasIamMixin:           true,
			hasOperationsMixin:    true,
			want:                  []string{"proto.pb.h"},
		},
		{
			name:                  "grpc with all mixins",
			generateGrpcTransport: true,
			protoGrpcHeaderPath:   "main.grpc.pb.h",
			protoHeaderPath:       "main.pb.h",
			hasLocationMixin:      true,
			hasIamMixin:           true,
			hasOperationsMixin:    true,
			want: []string{
				"google/cloud/location/locations.grpc.pb.h",
				"google/iam/v1/iam_policy.grpc.pb.h",
				"google/longrunning/operations.grpc.pb.h",
				"main.grpc.pb.h",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := idempotencyPolicyProtoIncludes(
				test.generateGrpcTransport,
				test.protoGrpcHeaderPath,
				test.protoHeaderPath,
				test.hasLocationMixin,
				test.hasIamMixin,
				test.hasOperationsMixin,
			)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_DetectServiceMixins(t *testing.T) {
	mLoc := api.NewTestMethod("GetLocation")
	mLoc.SourceServiceID = "google.cloud.location.Locations"
	mIam := api.NewTestMethod("GetIamPolicy")
	mIam.SourceServiceID = "google.iam.v1.IAMPolicy"
	mOps := api.NewTestMethod("ListOperations")
	mOps.SourceServiceID = "google.longrunning.Operations"

	for _, test := range []struct {
		name string
		svc  *api.Service
		want serviceMixins
	}{
		{
			name: "nil service",
			svc:  nil,
			want: serviceMixins{},
		},
		{
			name: "location and iam mixins",
			svc:  api.NewTestService("MixinsService").WithMethods(mLoc, mIam),
			want: serviceMixins{
				hasLocation:   true,
				hasIam:        true,
				hasOperations: false,
			},
		},
		{
			name: "operations mixin via non poller",
			svc:  api.NewTestService("OpsService").WithMethods(mOps),
			want: serviceMixins{
				hasLocation:   false,
				hasIam:        false,
				hasOperations: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := detectServiceMixins(test.svc)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(serviceMixins{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_ConnectionImplHeaderIncludes(t *testing.T) {
	got := connectionImplHeaderIncludes(
		"retry_traits.h",
		"stub.h",
		"connection.h",
		"idempotency_policy.h",
		"options.h",
	)
	want := []string{
		"connection.h",
		"idempotency_policy.h",
		"options.h",
		"retry_traits.h",
		"stub.h",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestHelper_RestStubProtoIncludes(t *testing.T) {
	for _, test := range []struct {
		name                 string
		additionalProtoFiles []string
		hasLocationMixin     bool
		hasIamMixin          bool
		hasOperationsMixin   bool
		hasLongrunningMethod bool
		protoHeaderPath      string
		want                 []string
	}{
		{
			name:                 "empty inputs",
			additionalProtoFiles: nil,
			hasLocationMixin:     false,
			hasIamMixin:          false,
			hasOperationsMixin:   false,
			hasLongrunningMethod: false,
			protoHeaderPath:      "",
			want:                 nil,
		},
		{
			name:                 "with mixins and lro and additional protos",
			additionalProtoFiles: []string{"google/extra/first.proto", "google/extra/first.proto"},
			hasLocationMixin:     true,
			hasIamMixin:          true,
			hasOperationsMixin:   true,
			hasLongrunningMethod: true,
			protoHeaderPath:      "google/service/main.pb.h",
			want: []string{
				"google/extra/first.pb.h",
				"google/cloud/location/locations.pb.h",
				"google/iam/v1/iam_policy.pb.h",
				"google/longrunning/operations.pb.h",
				"google/service/main.pb.h",
			},
		},
		{
			name:                 "lro alone includes operations proto",
			additionalProtoFiles: nil,
			hasLocationMixin:     false,
			hasIamMixin:          false,
			hasOperationsMixin:   false,
			hasLongrunningMethod: true,
			protoHeaderPath:      "",
			want:                 []string{"google/longrunning/operations.pb.h"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := restStubProtoIncludes(
				test.additionalProtoFiles,
				test.hasLocationMixin,
				test.hasIamMixin,
				test.hasOperationsMixin,
				test.hasLongrunningMethod,
				test.protoHeaderPath,
			)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_StubProtoIncludes_Direct(t *testing.T) {
	for _, test := range []struct {
		name                 string
		additionalProtoFiles []string
		hasLocationMixin     bool
		hasIamMixin          bool
		hasOperationsMixin   bool
		hasLongrunningMethod bool
		protoGrpcHeaderPath  string
		want                 []string
	}{
		{
			name:                 "empty inputs",
			additionalProtoFiles: nil,
			hasLocationMixin:     false,
			hasIamMixin:          false,
			hasOperationsMixin:   false,
			hasLongrunningMethod: false,
			protoGrpcHeaderPath:  "",
			want:                 nil,
		},
		{
			name:                 "all phases present with LRO without operations mixin",
			additionalProtoFiles: []string{"google/z.proto", "google/a.proto", "google/a.proto"},
			hasLocationMixin:     true,
			hasIamMixin:          true,
			hasOperationsMixin:   false,
			hasLongrunningMethod: true,
			protoGrpcHeaderPath:  "google/main.grpc.pb.h",
			want: []string{
				"google/a.pb.h",
				"google/z.pb.h",
				"google/cloud/location/locations.grpc.pb.h",
				"google/iam/v1/iam_policy.grpc.pb.h",
				"google/longrunning/operations.grpc.pb.h",
				"google/main.grpc.pb.h",
			},
		},
		{
			name:                 "operations mixin suppresses duplicate lro",
			additionalProtoFiles: nil,
			hasLocationMixin:     false,
			hasIamMixin:          false,
			hasOperationsMixin:   true,
			hasLongrunningMethod: true,
			protoGrpcHeaderPath:  "google/main.grpc.pb.h",
			want: []string{
				"google/longrunning/operations.grpc.pb.h",
				"google/main.grpc.pb.h",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := stubProtoIncludes(
				test.additionalProtoFiles,
				test.hasLocationMixin,
				test.hasIamMixin,
				test.hasOperationsMixin,
				test.hasLongrunningMethod,
				test.protoGrpcHeaderPath,
			)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_IsLongrunningPoller(t *testing.T) {
	mNonOps := api.NewTestMethod("GetOperation")
	mNonOps.SourceServiceID = "google.cloud.example.Service"

	mList := api.NewTestMethod("ListOperations")
	mList.SourceServiceID = "google.longrunning.Operations"

	mPoller := api.NewTestMethod("GetOperation")
	mPoller.SourceServiceID = "google.longrunning.Operations"

	mCancel := api.NewTestMethod("CancelOperation")
	mCancel.SourceServiceID = "google.longrunning.Operations"

	mWait := api.NewTestMethod("WaitOperation")
	mWait.SourceServiceID = "google.longrunning.Operations"

	mWithOpsBinding := api.NewTestMethod("GetOperation").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("name").WithLiteral("operations")))
	mWithOpsBinding.SourceServiceID = "google.longrunning.Operations"

	mWithOtherBinding := api.NewTestMethod("GetOperation").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("name").WithLiteral("locations")))
	mWithOtherBinding.SourceServiceID = "google.longrunning.Operations"

	for _, test := range []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{
			name:   "nil method",
			method: nil,
			want:   false,
		},
		{
			name:   "non operations service",
			method: mNonOps,
			want:   false,
		},
		{
			name:   "list operations is not poller",
			method: mList,
			want:   false,
		},
		{
			name:   "default get operation is poller",
			method: mPoller,
			want:   true,
		},
		{
			name:   "cancel operation is poller",
			method: mCancel,
			want:   true,
		},
		{
			name:   "wait operation is poller",
			method: mWait,
			want:   true,
		},
		{
			name:   "binding with operations segment is poller",
			method: mWithOpsBinding,
			want:   true,
		},
		{
			name:   "binding without operations segment is not poller",
			method: mWithOtherBinding,
			want:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isLongrunningPoller(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelper_PopulateServiceIncludes(t *testing.T) {
	t.Run("nil inputs safe handling", func(t *testing.T) {
		got := populateServiceIncludes(nil, nil, "google/cloud/test", nil, "2023")
		if got == nil {
			t.Fatal("expected non-nil serviceAnnotations")
		}
		if diff := cmp.Diff("2024", got.SourcesCcCopyrightYear); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("populates all include fields correctly", func(t *testing.T) {
		mLoc := api.NewTestMethod("GetLocation")
		mLoc.SourceServiceID = "google.cloud.location.Locations"
		svc := api.NewTestService("EchoService").WithMethods(mLoc)

		sAnn := &serviceAnnotations{
			Name:                        svc.Name,
			RetryTraitsHeaderPath:       "google/cloud/echo/retry_traits.h",
			IdempotencyPolicyHeaderPath: "google/cloud/echo/connection_idempotency_policy.h",
			OptionDefaultsHeaderPath:    "google/cloud/echo/internal/option_defaults.h",
			TracingConnectionHeaderPath: "google/cloud/echo/internal/tracing_connection.h",
			OptionsHeaderPath:           "google/cloud/echo/options.h",
			ConnectionImplHeaderPath:    "google/cloud/echo/internal/connection_impl.h",
			StubFactoryHeaderPath:       "google/cloud/echo/internal/stub_factory.h",
			StubHeaderPath:              "google/cloud/echo/internal/stub.h",
			ConnectionHeaderPath:        "google/cloud/echo/connection.h",
			ProtoHeaderPath:             "google/cloud/echo/echo.pb.h",
			ProtoGrpcHeaderPath:         "google/cloud/echo/echo.grpc.pb.h",
			GenerateGrpcTransport:       true,
			GenerateRestTransport:       false,
			GenerateRoundRobinDecorator: false,
			HasLongrunningMethod:        false,
		}

		sAnn = populateServiceIncludes(sAnn, svc, "google/cloud/echo", []string{"google/cloud/echo/extra.proto"}, "2023")

		want := &serviceAnnotations{
			Name:                        svc.Name,
			RetryTraitsHeaderPath:       "google/cloud/echo/retry_traits.h",
			IdempotencyPolicyHeaderPath: "google/cloud/echo/connection_idempotency_policy.h",
			OptionDefaultsHeaderPath:    "google/cloud/echo/internal/option_defaults.h",
			TracingConnectionHeaderPath: "google/cloud/echo/internal/tracing_connection.h",
			OptionsHeaderPath:           "google/cloud/echo/options.h",
			ConnectionImplHeaderPath:    "google/cloud/echo/internal/connection_impl.h",
			StubFactoryHeaderPath:       "google/cloud/echo/internal/stub_factory.h",
			StubHeaderPath:              "google/cloud/echo/internal/stub.h",
			ConnectionHeaderPath:        "google/cloud/echo/connection.h",
			ProtoHeaderPath:             "google/cloud/echo/echo.pb.h",
			ProtoGrpcHeaderPath:         "google/cloud/echo/echo.grpc.pb.h",
			GenerateGrpcTransport:       true,
			GenerateRestTransport:       false,
			GenerateRoundRobinDecorator: false,
			HasLongrunningMethod:        false,
			HasLocationMixin:            true,
			HasIamMixin:                 false,
			HasOperationsMixin:          false,
			HasOperationsStub:           false,
			SourcesCcCopyrightYear:      "2024",
			SourcesContext:              &sourcesContextAnnotation{UseSourcesYear: true},
			ConnectionHeaderIncludes: []string{
				"google/cloud/echo/connection_idempotency_policy.h",
				"google/cloud/echo/retry_traits.h",
			},
			ConnectionSourceIncludes: []string{
				"google/cloud/echo/internal/connection_impl.h",
				"google/cloud/echo/internal/option_defaults.h",
				"google/cloud/echo/internal/stub_factory.h",
				"google/cloud/echo/internal/tracing_connection.h",
				"google/cloud/echo/options.h",
			},
			ConnectionImplHeaderIncludes: []string{
				"google/cloud/echo/connection.h",
				"google/cloud/echo/connection_idempotency_policy.h",
				"google/cloud/echo/internal/stub.h",
				"google/cloud/echo/options.h",
				"google/cloud/echo/retry_traits.h",
			},
			SourcesCcIncludes: []string{
				"google/cloud/echo/echo_client.cc",
				"google/cloud/echo/echo_connection.cc",
				"google/cloud/echo/echo_connection_idempotency_policy.cc",
				"google/cloud/echo/internal/echo_auth_decorator.cc",
				"google/cloud/echo/internal/echo_connection_impl.cc",
				"google/cloud/echo/internal/echo_logging_decorator.cc",
				"google/cloud/echo/internal/echo_metadata_decorator.cc",
				"google/cloud/echo/internal/echo_option_defaults.cc",
				"google/cloud/echo/internal/echo_stub.cc",
				"google/cloud/echo/internal/echo_stub_factory.cc",
				"google/cloud/echo/internal/echo_tracing_connection.cc",
				"google/cloud/echo/internal/echo_tracing_stub.cc",
			},
			ConnectionProtoIncludes: []string{
				"google/cloud/echo/echo.pb.h",
				"google/cloud/echo/extra.pb.h",
			},
			IdempotencyPolicyProtoIncludes: []string{
				"google/cloud/echo/echo.grpc.pb.h",
				"google/cloud/location/locations.grpc.pb.h",
			},
			StubProtoIncludes: []string{
				"google/cloud/echo/extra.pb.h",
				"google/cloud/location/locations.grpc.pb.h",
				"google/cloud/echo/echo.grpc.pb.h",
			},
			RestStubProtoIncludes: []string{
				"google/cloud/echo/extra.pb.h",
				"google/cloud/location/locations.pb.h",
				"google/cloud/echo/echo.pb.h",
			},
		}
		if diff := cmp.Diff(want, sAnn, cmp.AllowUnexported(sourcesContextAnnotation{})); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestAnnotateService_ComputeLRO_RestStubProtoIncludes(t *testing.T) {
	op := api.NewTestMessage("Operation").WithPackage("google.cloud.cpp.compute.v1")
	req := api.NewTestMessage("InsertRequest").WithPackage("google.cloud.compute.v1")
	method := api.NewTestMethod("Insert").
		WithInput(req).
		WithOutput(op).
		WithOperationService("RegionOperations")
	svc := api.NewTestService("RegionOperationsService").
		WithPackage("google.cloud.compute.v1").
		WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc})
	model.DefinitionLocations = map[string]api.SourceLocation{
		svc.ID: {Filename: "google/cloud/compute/v1/region_operations.proto", Line: 1},
	}
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   []string{"// Sample Boilerplate"},
	}
	libCfg := &config.CppLibrary{GenerateRestTransport: true}
	c := newCodec(libCfg)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	if !slices.Contains(got.RestStubProtoIncludes, "google/cloud/compute/region_operations/v1/region_operations.pb.h") {
		t.Errorf("expected RestStubProtoIncludes to contain region_operations.pb.h, got %v", got.RestStubProtoIncludes)
	}
	if slices.Contains(got.RestStubProtoIncludes, "google/longrunning/operations.pb.h") {
		t.Errorf("expected RestStubProtoIncludes to NOT contain google/longrunning/operations.pb.h, got %v", got.RestStubProtoIncludes)
	}
}
