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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod_RoutingFieldTypes(t *testing.T) {
	extEnum := api.NewTestEnum("ExtMode").
		WithPackage("google.type").
		WithValues(
			api.NewTestEnumValue("EXT_MODE_UNSPECIFIED", 0),
			api.NewTestEnumValue("EXT_MODE_FAST", 1),
		)
	inputType := api.NewTestMessage("RouteRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("page_size").WithType(api.TypezInt32),
			api.NewTestField("ext_mode").WithEnumType(extEnum),
		)
	outputType := api.NewTestMessage("RouteResponse").
		WithPackage("google.cloud.test.v1")

	httpScalarMethod := api.NewTestMethod("GetTarget").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(
			api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").
				WithVariableNamed("page_size").
				Build(),
		)

	explicitExtMethod := api.NewTestMethod("RouteExternal").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithLiteral("route").Build()).
		WithRouting(
			api.NewTestRoutingInfo("mode", api.NewTestRoutingInfoVariant("ext_mode")),
			api.NewTestRoutingInfo("count", api.NewTestRoutingInfoVariant("page_size")),
		)

	service := api.NewTestService("TestService").
		WithPackage("google.cloud.test.v1").
		WithMethods(httpScalarMethod, explicitExtMethod)
	model := api.NewTestAPI([]*api.Message{inputType, outputType}, []*api.Enum{extEnum}, []*api.Service{service}).
		WithPackageName("google.cloud.test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name: "google-cloud-test-v1",
		Swift: swiftConfig(t, []config.SwiftDependency{
			{Name: "GoogleType", ApiPackage: "google.type"},
		}),
	}
	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "TestProtos",
	}
	c, err := newCodec(model, library, module, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotHTTP := httpScalarMethod.Codec().(*methodAnnotations)
	if len(gotHTTP.PathBindings) != 0 {
		t.Errorf("grpc-client PathBindings = %d, want 0", len(gotHTTP.PathBindings))
	}
	if gotHTTP.HTTPMethod != "GET" || !gotHTTP.Idempotent() {
		t.Errorf("got HTTPMethod=%q Idempotent()=%v, want GET / true", gotHTTP.HTTPMethod, gotHTTP.Idempotent())
	}
	wantHTTPParams := []*routingParam{
		{
			RoutingKey: "name",
			Variants: []*routingParamVariant{
				{
					FieldSteps:       []*routingFieldStep{{PropertyName: "name"}},
					MatchingSegments: []string{".multiWildcard"},
					Last:             true,
				},
			},
		},
		{
			RoutingKey: "page_size",
			Variants: []*routingParamVariant{
				{
					FieldSteps:       []*routingFieldStep{{PropertyName: "pageSize"}},
					ConvertToString:  true,
					MatchingSegments: []string{".multiWildcard"},
					Last:             true,
				},
			},
		},
	}
	if diff := cmp.Diff(wantHTTPParams, gotHTTP.RoutingParams); diff != "" {
		t.Errorf("httpScalarMethod.RoutingParams mismatch (-want +got):\n%s", diff)
	}

	svcAnn := service.Codec().(*serviceAnnotations)
	if diff := cmp.Diff([]string{"GoogleType"}, svcAnn.ServiceImports()); diff != "" {
		t.Errorf("service.ServiceImports() mismatch (-want +got):\n%s", diff)
	}

	gotExplicit := explicitExtMethod.Codec().(*methodAnnotations)
	wantExplicitParams := []*routingParam{
		{
			RoutingKey: "mode",
			Variants: []*routingParamVariant{
				{
					FieldSteps:       []*routingFieldStep{{PropertyName: "extMode"}},
					IsEnum:           true,
					RequiredPackages: []string{"google.type"},
					Last:             true,
				},
			},
		},
		{
			RoutingKey: "count",
			Variants: []*routingParamVariant{
				{
					FieldSteps:      []*routingFieldStep{{PropertyName: "pageSize"}},
					ConvertToString: true,
					Last:            true,
				},
			},
		},
	}
	if diff := cmp.Diff(wantExplicitParams, gotExplicit.RoutingParams); diff != "" {
		t.Errorf("explicitExtMethod.RoutingParams mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_HTTPClientSkipsRoutingParams(t *testing.T) {
	extEnum := api.NewTestEnum("ExtMode").
		WithPackage("google.type").
		WithValues(api.NewTestEnumValue("EXT_MODE_UNSPECIFIED", 0))
	inputType := api.NewTestMessage("HTTPRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("payload").WithType(api.TypezBytes),
			api.NewTestField("ext_mode").WithEnumType(extEnum),
		)
	outputType := api.NewTestMessage("HTTPResponse").WithPackage("google.cloud.test.v1")
	method := api.NewTestMethod("GetResource").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").Build()).
		WithRouting(
			api.NewTestRoutingInfo("payload", api.NewTestRoutingInfoVariant("payload")),
			api.NewTestRoutingInfo("mode", api.NewTestRoutingInfoVariant("ext_mode")),
		)
	service := api.NewTestService("HTTPService").
		WithPackage("google.cloud.test.v1").
		WithMethods(method)
	model := api.NewTestAPI([]*api.Message{inputType, outputType}, []*api.Enum{extEnum}, []*api.Service{service}).
		WithPackageName("google.cloud.test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	c := newTestCodec(t, model, &config.Library{
		Name: "google-cloud-test-v1",
		Swift: swiftConfig(t, []config.SwiftDependency{
			{Name: "GoogleType", ApiPackage: "google.type"},
		}),
	})
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	gotMethod := method.Codec().(*methodAnnotations)
	if len(gotMethod.RoutingParams) != 0 {
		t.Errorf("HTTP client RoutingParams = %v, want empty", gotMethod.RoutingParams)
	}
	svcAnn := service.Codec().(*serviceAnnotations)
	if len(svcAnn.ServiceImports()) != 0 {
		t.Errorf("HTTP client ServiceImports() = %v, want empty", svcAnn.ServiceImports())
	}
}

func TestAnnotateMethod_RoutingError(t *testing.T) {
	inputType := api.NewTestMessage("BadRouteRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("payload").WithType(api.TypezBytes),
		)
	outputType := api.NewTestMessage("BadRouteResponse").
		WithPackage("google.cloud.test.v1")

	for _, test := range []struct {
		name   string
		method *api.Method
	}{
		{
			name: "explicit routing references missing field",
			method: api.NewTestMethod("MissingExplicit").
				WithInput(inputType).
				WithOutput(outputType).
				WithRouting(api.NewTestRoutingInfo("missing", api.NewTestRoutingInfoVariant("does_not_exist"))),
		},
		{
			name: "implicit HTTP binding references bytes field",
			method: api.NewTestMethod("BytesHTTP").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("payload").Build()),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := api.NewTestService("TestService").
				WithPackage("google.cloud.test.v1").
				WithMethods(test.method)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service}).
				WithPackageName("google.cloud.test.v1")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			c, err := newCodec(model, &config.Library{Swift: swiftConfig(t, nil)}, &config.SwiftModule{
				ModuleType: "grpc-client",
				ModulePath: "TestProtos",
			}, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if gotErr := c.annotateModel(); gotErr == nil {
				t.Errorf("annotateModel() succeeded for %s, want error", test.name)
			}
		})
	}
}
