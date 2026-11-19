// Copyright 2025 Google LLC
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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethodNames(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		MethodID string
		Want     *methodAnnotation
	}{
		{
			MethodID: ".test.v1.ResourceService.move",
			Want: &methodAnnotation{
				Name:                "r#move",
				NameNoMangling:      "move",
				BuilderName:         "Move",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "crate::model::Response",
			},
		},
		{
			MethodID: ".test.v1.ResourceService.Delete",
			Want: &methodAnnotation{
				Name:                "delete",
				NameNoMangling:      "delete",
				BuilderName:         "Delete",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "()",
			},
		},
		{
			MethodID: ".test.v1.ResourceService.Self",
			Want: &methodAnnotation{
				Name:                "self_",
				NameNoMangling:      "self",
				BuilderName:         "Self_",
				Body:                "None::<gaxi::http::NoBody>",
				ServiceNameToPascal: "ResourceService",
				ServiceNameToCamel:  "resourceService",
				ServiceNameToSnake:  "resource_service",
				ReturnType:          "crate::model::Response",
			},
		},
	} {
		gotMethod := model.Method(test.MethodID)
		if gotMethod == nil {
			t.Errorf("missing method %s", test.MethodID)
			continue
		}
		got := gotMethod.Codec().(*methodAnnotation)
		if diff := cmp.Diff(test.Want, got, cmpopts.IgnoreFields(methodAnnotation{}, "PathInfo", "SystemParameters")); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestAnnotateDiscoveryAnnotations(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	methodID := ".test.v1.ResourceService.Delete"
	gotMethod := model.Method(methodID)
	if gotMethod == nil {
		t.Fatalf("missing method %s", methodID)
	}
	got := gotMethod.DiscoveryLro().Codec().(*discoveryLroAnnotations)
	want := &discoveryLroAnnotations{
		MethodName:          "delete",
		ReturnType:          "()",
		ServiceNameToPascal: "ResourceService",
		PollingPathParameters: []discoveryLroPathParameter{
			{Name: "project", SetterName: "project"},
			{Name: "zone", SetterName: "zone"},
			{Name: "r#type", SetterName: "type"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethodAPIVersion(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	// Inject an APIVersion to the existing model.
	methodID := ".test.v1.ResourceService.Delete"
	gotMethod := model.Method(methodID)
	if gotMethod == nil {
		t.Fatalf("missing method %s", methodID)
	}
	gotMethod.WithAPIVersion("v1_20260205")

	codec := newTestCodec(t, libconfig.SpecDiscovery, "", map[string]string{})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	got := gotMethod.Codec().(*methodAnnotation)
	want := []systemParameter{
		{Name: "$alt", Value: "json"},
		{Name: "$apiVersion", Value: "v1_20260205"},
	}
	if diff := cmp.Diff(want, got.SystemParameters); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethodInternalBuilders(t *testing.T) {
	model := annotateMethodModel(t)
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"internal-builders": "true",
	})
	_, err = annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	methodID := ".test.v1.ResourceService.Delete"
	gotMethod := model.Method(methodID)
	if gotMethod == nil {
		t.Fatalf("missing method %s", methodID)
	}
	got := gotMethod.Codec().(*methodAnnotation)
	if !got.InternalBuilders {
		t.Errorf("expected InternalBuilders to be true for method %s", methodID)
	}
	if got.BuilderVisibility() != "pub(crate)" {
		t.Errorf("mismatch in BuilderVisibility, want=pub(crate), got=%s", got.BuilderVisibility())
	}
}

func annotateMethodModel(t *testing.T) *api.API {
	t.Helper()
	request := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("project").WithType(api.TypezString),
			api.NewTestField("zone").WithType(api.TypezString),
			api.NewTestField("type").WithType(api.TypezString),
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("location").WithType(api.TypezString),
			api.NewTestField("cluster").WithType(api.TypezString),
		)
	response := api.NewTestMessage("Response").
		WithPackage("test.v1")
	empty := api.NewTestMessage("Empty").
		WithPackage("google.protobuf")

	methodMove := api.NewTestMethod("move").
		WithInput(request).
		WithOutput(response).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())

	methodDelete := api.NewTestMethod("Delete").
		WithInput(request).
		WithOutput(empty).
		WithVerb("DELETE").
		WithPathTemplate(api.NewPathTemplateBuilder().
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("zones").
			WithVariableNamed("zone").
			// This is unlikely, but want to test variables that
			// are reserved words.
			WithLiteral("types").
			WithVariableNamed("type").
			Build()).
		WithDiscoveryLro(api.NewDiscoveryLroBuilder().
			WithPollingPathParameters("project", "zone", "type").
			Build()).
		ReturnEmpty()

	methodSelf := api.NewTestMethod("Self").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())

	service := api.NewTestService("ResourceService").
		WithPackage("test.v1").
		WithMethods(methodMove, methodDelete, methodSelf)

	model := api.NewTestAPI(
		[]*api.Message{request, response},
		nil,
		[]*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	return model
}

func TestAnnotateMethodResourceNameTemplate(t *testing.T) {
	model := annotateMethodModel(t)

	// Helper to inject TargetResource
	injectTargetResource := func(methodID string, template string, fields [][]string) {
		m := model.Method(methodID)
		if m == nil {
			t.Fatalf("missing method %s", methodID)
		}
		if m.PathInfo() != nil && len(m.PathInfo().Bindings()) > 0 {
			m.WithBindings(api.NewPathBindingBuilder().
				WithVerb(m.PathInfo().Bindings()[0].Verb()).
				WithQueryParameterMap(m.PathInfo().Bindings()[0].QueryParameters()).
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("projects").
					WithVariableNamed("project").
					WithLiteral("zones").
					WithVariableNamed("zone").
					WithLiteral("types").
					WithVariableNamed("type").
					Build()).
				WithTargetResource(api.NewTargetResource(
					fields,
					api.ParseTemplateForTest(template),
				)).
				Build())
		}
	}

	// Setup: Inject resource for the "move" method
	injectTargetResource(".test.v1.ResourceService.move", "//Test.googleapis.com/projects/{project}/zones/{zone}/types/{type}", [][]string{
		{"project"},
		{"zone"},
		{"type"},
	})

	// Setup: Inject multiple bindings for the "Self" method
	mSelf := model.Method(".test.v1.ResourceService.Self")
	mSelf.WithBindings(
		api.NewPathBindingBuilder().
			WithVerb("GET").
			WithPathTemplate(api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithVariableNamed("name").
				Build()).
			WithTargetResource(api.NewTargetResource(
				[][]string{{"name"}},
				api.ParseTemplateForTest("//Test.googleapis.com/projects/{project}/locations/{location}/clusters/{cluster}"),
			)).
			Build(),
		api.NewPathBindingBuilder().
			WithVerb("GET").
			WithPathTemplate(api.NewPathTemplateBuilder().
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("locations").
				WithVariableNamed("location").
				WithLiteral("clusters").
				WithVariableNamed("cluster").
				Build()).
			WithTargetResource(api.NewTargetResource(
				[][]string{{"project"}, {"location"}, {"cluster"}},
				api.ParseTemplateForTest("//Test.googleapis.com/projects/{project}/locations/{location}/clusters/{cluster}"),
			)).
			Build(),
		api.NewPathBindingBuilder().
			WithVerb("GET").
			WithPathTemplate(api.NewPathTemplateBuilder().Build()).
			Build(),
	)

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name         string
		id           string
		want         *methodAnnotation
		wantBindings []*pathBindingAnnotation
	}{
		{
			name: "WithTargetResource",
			id:   ".test.v1.ResourceService.move",
			want: &methodAnnotation{
				HasResourceNameGeneration: true,
			},
		},
		{
			name: "WithoutTargetResource",
			id:   ".test.v1.ResourceService.Delete",
			want: &methodAnnotation{
				HasResourceNameGeneration: false,
			},
		},
		{
			name: "MultipleBindings",
			id:   ".test.v1.ResourceService.Self",
			want: &methodAnnotation{
				HasResourceNameGeneration: true,
			},
			wantBindings: []*pathBindingAnnotation{
				{
					HasResourceNameGeneration: true,
					ResourceNameTemplate:      "//Test.googleapis.com/projects/{}/locations/{}/clusters/{}",
					ResourceNameArgs:          []string{"var_name"},
				},
				{
					HasResourceNameGeneration: true,
					ResourceNameTemplate:      "//Test.googleapis.com/projects/{}/locations/{}/clusters/{}",
					ResourceNameArgs:          []string{"var_project", "var_location", "var_cluster"},
				},
				{
					HasResourceNameGeneration: true,
					ResourceNameTemplate:      "",
					ResourceNameArgs:          nil,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := model.Method(test.id)
			if m == nil {
				t.Fatalf("missing method %s", test.id)
			}
			got := m.Codec().(*methodAnnotation)
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(methodAnnotation{},
				"Name", "NameNoMangling", "BuilderName", "Body", "DocLines",
				"ServiceNameToPascal", "ServiceNameToCamel", "ServiceNameToSnake",
				"SystemParameters", "ReturnType", "PathInfo", "Attributes",
				"RoutingRequired", "DetailedTracingAttributes",
				"ResourceNameTemplateGrpc", "GrpcResourceNameArgs",
				"InternalBuilders")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if test.wantBindings != nil {
				for i, wantBinding := range test.wantBindings {
					gotBinding := m.PathInfo().Bindings()[i].Codec().(*pathBindingAnnotation)
					if diff := cmp.Diff(wantBinding, gotBinding, cmpopts.IgnoreFields(pathBindingAnnotation{}, "DetailedTracingAttributes", "PathFmt", "QueryParams", "Substitutions")); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestFormatResourceNameTemplateFromPath(t *testing.T) {
	method := api.NewTestMethod("method")
	for _, test := range []struct {
		name    string
		binding *api.PathBinding
		want    string
		wantErr bool
	}{
		{
			name: "Basic",
			binding: api.NewPathBindingBuilder().
				WithTargetResource(api.NewTargetResource(
					nil,
					api.ParseTemplateForTest("//test.googleapis.com/projects/{project}/zones/{zone}"),
				)).
				Build(),
			want: "//test.googleapis.com/projects/{}/zones/{}",
		},
		{
			name: "With Extended Field Path",
			binding: api.NewPathBindingBuilder().
				WithTargetResource(api.NewTargetResource(
					nil,
					api.ParseTemplateForTest("//test.googleapis.com/items/{item.id}"),
				)).
				Build(),
			want: "//test.googleapis.com/items/{}",
		},
		{
			name: "Discovery API Compute V1 Example",
			binding: api.NewPathBindingBuilder().
				WithPathTemplate(api.NewPathTemplateBuilder().
					WithLiteral("compute").
					WithLiteral("v1").
					WithLiteral("projects").
					WithVariableNamed("project").
					WithLiteral("zones").
					WithVariableNamed("zone").
					Build()).
				WithTargetResource(api.NewTargetResource(
					nil,
					// Notice that constructTemplate already stripped out "compute/v1"
					api.ParseTemplateForTest("//compute.googleapis.com/projects/{project}/zones/{zone}"),
				)).
				Build(),
			want: "//compute.googleapis.com/projects/{}/zones/{}",
		},
		{
			name:    "Missing TargetResource",
			binding: api.NewPathBindingBuilder().Build(),
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatResourceNameTemplateFromPath(method, test.binding)
			if (err != nil) != test.wantErr {
				t.Fatalf("formatResourceNameTemplateFromPath() error = %v, wantErr %v", err, test.wantErr)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateSampleInfo(t *testing.T) {
	field := api.NewTestField("name").
		WithType(api.TypezString).
		WithResourceNamePattern(api.NewTestResourceNamePattern(
			api.NewResourceNameSegment("projects", ""),
			api.NewResourceNameSegment("", "project"),
		))
	message := api.NewTestMessage("TestMessage").
		WithPackage("test.v1").
		WithFields(field)
	method := api.NewTestMethod("TestMethod")
	si := api.NewSampleInfoBuilder().
		WithResourceNameField(field).
		Build()

	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{})
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

	codec.annotateSampleInfo(si, method)

	got := si.Codec().(*sampleInfoAnnotation)
	want := &sampleInfoAnnotation{
		StringParameters:   []string{"project_id"},
		FormatResourceName: true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMethodAnnotationsDetailedTracing(t *testing.T) {
	model := serviceAnnotationsModel()
	method := model.Method(".test.v1.ResourceService.GetResource")
	if method == nil {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	annotateModel(model, codec)
	got := method.Codec().(*methodAnnotation)
	if !got.DetailedTracingAttributes {
		t.Errorf("methodAnnotation.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestIsBidiStreaming(t *testing.T) {
	for _, test := range []struct {
		name   string
		client bool
		server bool
		isGrpc bool
		want   bool
	}{
		{"bidi streaming with gRPC", true, true, true, true},
		{"bidi streaming without gRPC", true, true, false, false},
		{"client-only streaming", true, false, true, false},
		{"server-only streaming", false, true, true, false},
		{"unary", false, false, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ann := &methodAnnotation{
				ClientSideStreaming: test.client,
				ServerSideStreaming: test.server,
				IsGrpc:              test.isGrpc,
			}
			if got := ann.IsBidiStreaming(); got != test.want {
				t.Errorf("methodAnnotation.IsBidiStreaming() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsServerStreaming(t *testing.T) {
	for _, test := range []struct {
		name   string
		client bool
		server bool
		isGrpc bool
		want   bool
	}{
		{"bidi streaming", true, true, true, false},
		{"client-only streaming", true, false, true, false},
		{"server-only streaming with gRPC", false, true, true, true},
		{"server-only streaming without gRPC", false, true, false, false},
		{"unary", false, false, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ann := &methodAnnotation{
				ClientSideStreaming: test.client,
				ServerSideStreaming: test.server,
				IsGrpc:              test.isGrpc,
			}
			if got := ann.IsServerStreaming(); got != test.want {
				t.Errorf("methodAnnotation.IsServerStreaming() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsHttp(t *testing.T) {
	for _, test := range []struct {
		name   string
		isGrpc bool
		want   bool
	}{
		{"gRPC method", true, false},
		{"HTTP method", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ann := &methodAnnotation{IsGrpc: test.isGrpc}
			if got := ann.IsHttp(); got != test.want {
				t.Errorf("methodAnnotation.IsHttp() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMethodUsesGrpc(t *testing.T) {
	msg := api.NewTestMessage("Message").WithPackage("test.v1")
	unaryMethod := api.NewTestMethod("Unary").WithInput(msg).WithOutput(msg).WithPathTemplate(api.NewPathTemplateBuilder().Build())
	bidiMethod := api.NewTestMethod("Bidi").WithInput(msg).WithOutput(msg).WithBidiStreaming()
	serverMethod := api.NewTestMethod("Server").WithInput(msg).WithOutput(msg).WithServerSideStreaming()
	clientMethod := api.NewTestMethod("Client").WithInput(msg).WithOutput(msg).WithClientSideStreaming()

	lroMethod := api.NewTestMethod("Lro").WithInput(msg).WithOutput(msg).WithOperationInfo(api.NewTestOperationInfo(msg.ID(), msg.ID()))

	for _, test := range []struct {
		name             string
		method           *api.Method
		templateOverride string
		options          map[string]string
		want             bool
	}{
		{
			name:    "unary method defaults to HTTP",
			method:  unaryMethod,
			options: map[string]string{},
			want:    false,
		},
		{
			name:    "bidi streaming by default",
			method:  bidiMethod,
			options: map[string]string{},
			want:    true,
		},
		{
			name:    "server streaming by default",
			method:  serverMethod,
			options: map[string]string{},
			want:    true,
		},
		{
			name:    "client streaming not included by default",
			method:  clientMethod,
			options: map[string]string{},
			want:    false,
		},
		{
			name:   "client streaming with include-streaming-methods",
			method: clientMethod,
			options: map[string]string{
				"include-streaming-methods": "true",
			},
			want: true,
		},
		{
			name:             "template without grpc ignores streaming",
			method:           bidiMethod,
			templateOverride: "templates/http-client",
			options:          map[string]string{},
			want:             false,
		},
		{
			name:   "LRO method defaults to HTTP on default template with default_transport grpc",
			method: lroMethod,
			options: map[string]string{
				"default-transport": "grpc",
			},
			want: false,
		},
		{
			name:             "LRO method uses gRPC on grpc-client template",
			method:           lroMethod,
			templateOverride: "templates/grpc-client",
			options: map[string]string{
				"default-transport": "grpc",
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			c.templateOverride = test.templateOverride
			if got := c.methodUsesGrpc(test.method); got != test.want {
				t.Errorf("methodUsesGrpc() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAnnotateMethodIdempotencyHook(t *testing.T) {
	// 1. When idempotency-hook is configured:
	modelWithHook := annotateMethodModel(t)
	if err := api.CrossReference(modelWithHook); err != nil {
		t.Fatal(err)
	}
	codecWithHook := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
		"idempotency-hook":          "resolve_idempotency",
	})
	if _, err := annotateModel(modelWithHook, codecWithHook); err != nil {
		t.Fatal(err)
	}
	methodWithHook := modelWithHook.Method(".test.v1.ResourceService.Delete")
	if methodWithHook == nil {
		t.Fatalf("missing method .test.v1.ResourceService.Delete")
	}
	got := methodWithHook.Codec().(*methodAnnotation)
	if !got.HasIdempotencyHook {
		t.Errorf("got.HasIdempotencyHook = false, want true")
	}
	if got.IdempotencyHook != "resolve_idempotency" {
		t.Errorf("got.IdempotencyHook = %q, want %q", got.IdempotencyHook, "resolve_idempotency")
	}

	// 2. When idempotency-hook is NOT configured:
	modelWithoutHook := annotateMethodModel(t)
	if err := api.CrossReference(modelWithoutHook); err != nil {
		t.Fatal(err)
	}
	codecWithoutHook := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	if _, err := annotateModel(modelWithoutHook, codecWithoutHook); err != nil {
		t.Fatal(err)
	}
	methodWithoutHook := modelWithoutHook.Method(".test.v1.ResourceService.Delete")
	if methodWithoutHook == nil {
		t.Fatalf("missing method .test.v1.ResourceService.Delete")
	}
	gotDefault := methodWithoutHook.Codec().(*methodAnnotation)
	if gotDefault.HasIdempotencyHook {
		t.Errorf("gotDefault.HasIdempotencyHook = true, want false")
	}
	if gotDefault.IdempotencyHook != "" {
		t.Errorf("gotDefault.IdempotencyHook = %q, want empty", gotDefault.IdempotencyHook)
	}
}

func TestAnnotateMethodExtensibleBuilders(t *testing.T) {
	model := annotateMethodModel(t)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
		"extensible-builders":       ".test.v1.ResourceService.Delete",
	})
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

	deleteMethod := model.Method(".test.v1.ResourceService.Delete")
	if deleteMethod == nil {
		t.Fatal("missing method .test.v1.ResourceService.Delete")
	}
	if !deleteMethod.Codec().(*methodAnnotation).ExtensibleBuilder {
		t.Error("expected ExtensibleBuilder to be true for .test.v1.ResourceService.Delete")
	}

	moveMethod := model.Method(".test.v1.ResourceService.move")
	if moveMethod == nil {
		t.Fatal("missing method .test.v1.ResourceService.move")
	}
	if moveMethod.Codec().(*methodAnnotation).ExtensibleBuilder {
		t.Error("expected ExtensibleBuilder to be false for .test.v1.ResourceService.move")
	}
}

func TestAnnotateMethodRoutingSegments(t *testing.T) {
	request := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("resource").WithType(api.TypezString),
			api.NewTestField("table_name").WithType(api.TypezString),
		)
	response := api.NewTestMessage("Response").WithPackage("test.v1")

	newMethod := func(name, routingName, field string, prefix, matching, suffix []string) *api.Method {
		return api.NewTestMethod(name).
			WithInput(request).
			WithOutput(response).
			WithVerb("POST").
			WithPathTemplate(api.NewPathTemplateBuilder().Build()).
			WithRouting(api.NewRoutingInfoBuilder().
				WithName(routingName).
				WithVariant(api.NewRoutingInfoVariantBuilder().
					WithFieldPath(field).
					WithPrefixSegments(prefix...).
					WithMatchingSegments(matching...).
					WithSuffixSegments(suffix...).
					Build()).
				Build())
	}

	methods := []*api.Method{
		// routing.proto Example 6a / 6b: projects/*/{instance_id=instances/*}/**
		newMethod("Example6a", "instance_id", "table_name",
			[]string{"projects", "*"}, []string{"instances", "*"}, []string{"**"}),
		// routing.proto Example 6b: {project_id=projects/*}/**
		newMethod("Example6b", "project_id", "table_name",
			nil, []string{"projects", "*"}, []string{"**"}),
		// Prefix + multi-segment matching without suffix: projects/*/{table_name=instances/*/tables/*}
		newMethod("MultiSegmentMatching", "table_name", "table_name",
			[]string{"projects", "*"}, []string{"instances", "*", "tables", "*"}, nil),
		// Non-singleton matching ending in **: {parent=projects/**}
		newMethod("MatchingTrailingWildcard", "parent", "name",
			nil, []string{"projects", "**"}, nil),
		// routing.proto Example 9 (prefix empty, literal + wildcard suffix): {table_location=regions/*/zones/*}/tables/*
		newMethod("Example9Suffix", "table_location", "table_name",
			nil, []string{"regions", "*", "zones", "*"}, []string{"tables", "*"}),
		// routing.proto Example 9 (both prefix and suffix): projects/*/{table_location=instances/*}/tables/*
		newMethod("Example9Both", "table_location", "table_name",
			[]string{"projects", "*"}, []string{"instances", "*"}, []string{"tables", "*"}),
		// Prefix ending in a literal: projects/{project=*}
		newMethod("PrefixLiteral", "project", "name",
			[]string{"projects"}, []string{"*"}, nil),
		// Multi-literal prefix: v1/projects/{project=*}
		newMethod("PrefixMultiLiteral", "project", "name",
			[]string{"v1", "projects"}, []string{"*"}, nil),
		// Literal-only suffix: {parent=projects/*}/tables
		newMethod("SuffixSingleLiteral", "parent", "name",
			nil, []string{"projects", "*"}, []string{"tables"}),
		// Multi-literal-only suffix: {parent=projects/*}/tables/subtables
		newMethod("SuffixMultiLiteral", "parent", "name",
			nil, []string{"projects", "*"}, []string{"tables", "subtables"}),
		// Suffix starting with single wildcard: {parent=projects/*}/*
		newMethod("SuffixSingleWildcard", "parent", "name",
			nil, []string{"projects", "*"}, []string{"*"}),
		// Suffix starting with single wildcard followed by **: {parent=projects/*}/*/**
		newMethod("SuffixWildcardAndMulti", "parent", "name",
			nil, []string{"projects", "*"}, []string{"*", "**"}),
		// StorageControl folder/IAM variant: {bucket=projects/*/buckets/*}/**
		newMethod("StorageFolder", "bucket", "name",
			nil, []string{"projects", "*", "buckets", "*"}, []string{"**"}),
		// StorageControl IAM variant: {bucket=projects/*/buckets/*}/objects/**
		newMethod("StorageObjects", "bucket", "resource",
			nil, []string{"projects", "*", "buckets", "*"}, []string{"objects", "**"}),
		// StorageControl IAM variant: {bucket=projects/*/buckets/*}/managedFolders/**
		newMethod("StorageManagedFolders", "bucket", "resource",
			nil, []string{"projects", "*", "buckets", "*"}, []string{"managedFolders", "**"}),
		// Fallback full wildcard: {bucket=**}
		newMethod("StorageFallback", "bucket", "resource",
			nil, []string{"**"}, nil),
	}

	service := api.NewTestService("TestService").
		WithPackage("test.v1").
		WithMethods(methods...)
	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
	})
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		methodID     string
		field        string
		wantPrefix   []string
		wantMatching []string
		wantSuffix   []string
	}{
		{
			methodID:     ".test.v1.TestService.Example6a",
			field:        "table_name",
			wantPrefix:   []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/")`},
			wantMatching: []string{`Segment::Literal("instances/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{"Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.Example6b",
			field:        "table_name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{"Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.MultiSegmentMatching",
			field:        "table_name",
			wantPrefix:   []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/")`},
			wantMatching: []string{`Segment::Literal("instances/")`, "Segment::SingleWildcard", `Segment::Literal("/tables/")`, "Segment::SingleWildcard"},
			wantSuffix:   nil,
		},
		{
			methodID:     ".test.v1.TestService.MatchingTrailingWildcard",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects")`, "Segment::TrailingMultiWildcard"},
			wantSuffix:   nil,
		},
		{
			methodID:     ".test.v1.TestService.Example9Suffix",
			field:        "table_name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("regions/")`, "Segment::SingleWildcard", `Segment::Literal("/zones/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/tables/")`, "Segment::SingleWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.Example9Both",
			field:        "table_name",
			wantPrefix:   []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/")`},
			wantMatching: []string{`Segment::Literal("instances/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/tables/")`, "Segment::SingleWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.PrefixLiteral",
			field:        "name",
			wantPrefix:   []string{`Segment::Literal("projects/")`},
			wantMatching: []string{"Segment::SingleWildcard"},
			wantSuffix:   nil,
		},
		{
			methodID:     ".test.v1.TestService.PrefixMultiLiteral",
			field:        "name",
			wantPrefix:   []string{`Segment::Literal("v1/projects/")`},
			wantMatching: []string{"Segment::SingleWildcard"},
			wantSuffix:   nil,
		},
		{
			methodID:     ".test.v1.TestService.SuffixSingleLiteral",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/tables")`},
		},
		{
			methodID:     ".test.v1.TestService.SuffixMultiLiteral",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/tables/subtables")`},
		},
		{
			methodID:     ".test.v1.TestService.SuffixSingleWildcard",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/")`, "Segment::SingleWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.SuffixWildcardAndMulti",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/")`, "Segment::SingleWildcard", "Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.StorageFolder",
			field:        "name",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/buckets/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{"Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.StorageObjects",
			field:        "resource",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/buckets/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/objects")`, "Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.StorageManagedFolders",
			field:        "resource",
			wantPrefix:   nil,
			wantMatching: []string{`Segment::Literal("projects/")`, "Segment::SingleWildcard", `Segment::Literal("/buckets/")`, "Segment::SingleWildcard"},
			wantSuffix:   []string{`Segment::Literal("/managedFolders")`, "Segment::TrailingMultiWildcard"},
		},
		{
			methodID:     ".test.v1.TestService.StorageFallback",
			field:        "resource",
			wantPrefix:   nil,
			wantMatching: []string{"Segment::MultiWildcard"},
			wantSuffix:   nil,
		},
	} {
		t.Run(test.methodID, func(t *testing.T) {
			m := model.Method(test.methodID)
			if m == nil {
				t.Fatalf("missing method %s", test.methodID)
			}
			got := m.Routing()[0].Variants()[0].Codec().(*routingVariantAnnotations)
			want := &routingVariantAnnotations{
				FieldAccessors:   []string{".map(|m| &m." + test.field + ")", ".map(|s| s.as_str())"},
				PrefixSegments:   test.wantPrefix,
				MatchingSegments: test.wantMatching,
				SuffixSegments:   test.wantSuffix,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
