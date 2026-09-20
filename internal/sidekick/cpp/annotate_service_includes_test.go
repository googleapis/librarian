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
		WithOutput(api.NewTestMessage("Response"))
	method.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{
			{
				Verb: "POST",
				PathTemplate: (&api.PathTemplate{}).
					WithLiteral("v1").
					WithLiteral("echo"),
			},
		},
	}
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

func TestAnnotateService_HasMapAndDuration(t *testing.T) {
	mapField := api.NewTestField("labels").WithMap()
	reqWithMap := api.NewTestMessage("RequestWithMap").WithFields(mapField)
	durationParam := api.NewTestField("timeout").WithType(api.TypezMessage).WithMessageType(
		api.NewTestMessage("Duration").WithPackage("google.protobuf"),
	)
	reqWithDuration := api.NewTestMessage("RequestWithDuration").WithFields(durationParam)

	mMap := api.NewTestMethod("CallWithMap").
		WithInput(reqWithMap).
		WithOutput(api.NewTestMessage("Resp1"))
	mDuration := api.NewTestMethod("CallWithDuration").
		WithInput(reqWithDuration).
		WithOutput(api.NewTestMessage("Resp2")).
		WithSignatures(&api.MethodSignature{
			Fields: []*api.Field{durationParam},
		})

	svc := api.NewTestService("FeatureService").
		WithMethods(mMap, mDuration)
	model := api.NewTestAPI([]*api.Message{reqWithMap, reqWithDuration}, nil, []*api.Service{svc})

	c := newCodec(nil)
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   []string{"// Sample Boilerplate"},
	}
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	if !got.HasMap {
		t.Errorf("expected HasMap true, got false")
	}
	if !got.HasDuration {
		t.Errorf("expected HasDuration true, got false")
	}
}

func TestAnnotateService_AuthorityEnvVar(t *testing.T) {
	svc := api.NewTestService("EchoService")
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	c := newCodec(&config.CppLibrary{
		ServiceEndpointEnvVar: "TEST_SERVICE_ENDPOINT",
	})
	modelAnn := &modelAnnotations{
		CopyrightYear: "2026",
		BoilerPlate:   []string{"// Sample Boilerplate"},
	}
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got := svc.Codec.(*serviceAnnotations)
	if diff := cmp.Diff("TEST_SERVICE_AUTHORITY", got.ServiceAuthorityEnvVar); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_RestAndRoundRobin(t *testing.T) {
	method := api.NewTestMethod("Foo")
	method.APIVersion = "2024-01-01"
	svc := api.NewTestService("DeprecatedService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{svc})
	model.DefinitionLocations = map[string]api.SourceLocation{
		svc.ID: {Filename: "generator/integration_tests/test_deprecated.proto", Line: 42},
	}
	modelAnn := &modelAnnotations{
		CopyrightYear: "2024",
		BoilerPlate:   license.HeaderBulk(),
	}
	boolTrue := true
	libCfg := &config.CppLibrary{
		ProductPath:                 "generator/integration_tests/golden/v1",
		GenerateRestTransport:       true,
		GenerateGrpcTransport:       &boolTrue,
		GenerateRoundRobinDecorator: true,
		EndpointLocationStyle:       "LOCATION_OPTIONALLY_DEPENDENT",
	}
	c := newCodec(libCfg)
	if err := c.annotateService(svc, modelAnn, model); err != nil {
		t.Fatal(err)
	}
	got, ok := svc.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("expected *serviceAnnotations, got %T", svc.Codec)
	}

	if !got.GenerateRestTransport {
		t.Errorf("expected GenerateRestTransport to be true")
	}
	if !got.GenerateGrpcTransport {
		t.Errorf("expected GenerateGrpcTransport to be true")
	}
	if !got.GenerateRoundRobinDecorator {
		t.Errorf("expected GenerateRoundRobinDecorator to be true")
	}
	if !got.IsLocationOptionallyDependent {
		t.Errorf("expected IsLocationOptionallyDependent to be true")
	}
	if !got.HasApiVersion {
		t.Errorf("expected HasApiVersion to be true")
	}
	if got.ApiVersion != "2024-01-01" {
		t.Errorf("expected ApiVersion to be 2024-01-01, got %q", got.ApiVersion)
	}

	wantPaths := map[string]string{
		"RestConnectionHeaderPath":        "generator/integration_tests/golden/v1/deprecated_rest_connection.h",
		"RestConnectionImplHeaderPath":    "generator/integration_tests/golden/v1/internal/deprecated_rest_connection_impl.h",
		"RestStubHeaderPath":              "generator/integration_tests/golden/v1/internal/deprecated_rest_stub.h",
		"RestStubFactoryHeaderPath":       "generator/integration_tests/golden/v1/internal/deprecated_rest_stub_factory.h",
		"RestLoggingDecoratorHeaderPath":  "generator/integration_tests/golden/v1/internal/deprecated_rest_logging_decorator.h",
		"RestMetadataDecoratorHeaderPath": "generator/integration_tests/golden/v1/internal/deprecated_rest_metadata_decorator.h",
		"RoundRobinHeaderPath":            "generator/integration_tests/golden/v1/internal/deprecated_round_robin_decorator.h",
	}
	gotPaths := map[string]string{
		"RestConnectionHeaderPath":        got.RestConnectionHeaderPath,
		"RestConnectionImplHeaderPath":    got.RestConnectionImplHeaderPath,
		"RestStubHeaderPath":              got.RestStubHeaderPath,
		"RestStubFactoryHeaderPath":       got.RestStubFactoryHeaderPath,
		"RestLoggingDecoratorHeaderPath":  got.RestLoggingDecoratorHeaderPath,
		"RestMetadataDecoratorHeaderPath": got.RestMetadataDecoratorHeaderPath,
		"RoundRobinHeaderPath":            got.RoundRobinHeaderPath,
	}
	if diff := cmp.Diff(wantPaths, gotPaths); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantGuards := map[string]string{
		"RestConnectionHeaderIncludeGuard":        "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_DEPRECATED_REST_CONNECTION_H",
		"RestConnectionImplHeaderIncludeGuard":    "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_REST_CONNECTION_IMPL_H",
		"RestStubHeaderIncludeGuard":              "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_REST_STUB_H",
		"RestStubFactoryHeaderIncludeGuard":       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_REST_STUB_FACTORY_H",
		"RestLoggingDecoratorHeaderIncludeGuard":  "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_REST_LOGGING_DECORATOR_H",
		"RestMetadataDecoratorHeaderIncludeGuard": "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_REST_METADATA_DECORATOR_H",
		"RoundRobinHeaderIncludeGuard":            "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_INTERNAL_DEPRECATED_ROUND_ROBIN_DECORATOR_H",
	}
	gotGuards := map[string]string{
		"RestConnectionHeaderIncludeGuard":        got.RestConnectionHeaderIncludeGuard,
		"RestConnectionImplHeaderIncludeGuard":    got.RestConnectionImplHeaderIncludeGuard,
		"RestStubHeaderIncludeGuard":              got.RestStubHeaderIncludeGuard,
		"RestStubFactoryHeaderIncludeGuard":       got.RestStubFactoryHeaderIncludeGuard,
		"RestLoggingDecoratorHeaderIncludeGuard":  got.RestLoggingDecoratorHeaderIncludeGuard,
		"RestMetadataDecoratorHeaderIncludeGuard": got.RestMetadataDecoratorHeaderIncludeGuard,
		"RoundRobinHeaderIncludeGuard":            got.RoundRobinHeaderIncludeGuard,
	}
	if diff := cmp.Diff(wantGuards, gotGuards); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantClasses := map[string]string{
		"RestStubClassName":                 "DeprecatedServiceRestStub",
		"DefaultRestStubClassName":          "DefaultDeprecatedServiceRestStub",
		"RestLoggingDecoratorClassName":     "DeprecatedServiceRestLogging",
		"RestMetadataDecoratorClassName":    "DeprecatedServiceRestMetadata",
		"RestConnectionImplClassName":       "DeprecatedServiceRestConnectionImpl",
		"RoundRobinClassName":               "DeprecatedServiceRoundRobin",
		"MakeRestConnectionFunctionName":    "MakeDeprecatedServiceConnectionRest",
		"CreateDefaultRestStubFunctionName": "CreateDefaultDeprecatedServiceRestStub",
	}
	gotClasses := map[string]string{
		"RestStubClassName":                 got.RestStubClassName,
		"DefaultRestStubClassName":          got.DefaultRestStubClassName,
		"RestLoggingDecoratorClassName":     got.RestLoggingDecoratorClassName,
		"RestMetadataDecoratorClassName":    got.RestMetadataDecoratorClassName,
		"RestConnectionImplClassName":       got.RestConnectionImplClassName,
		"RoundRobinClassName":               got.RoundRobinClassName,
		"MakeRestConnectionFunctionName":    got.MakeRestConnectionFunctionName,
		"CreateDefaultRestStubFunctionName": got.CreateDefaultRestStubFunctionName,
	}
	if diff := cmp.Diff(wantClasses, gotClasses); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
