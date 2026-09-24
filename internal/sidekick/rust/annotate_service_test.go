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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestServiceAnnotationsExtendGrpcTransport(t *testing.T) {
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"extend-grpc-transport": "true",
	})
	annotateModel(model, codec)
	serviceAnn := service.Codec.(*serviceAnnotations)

	if !serviceAnn.ExtendGrpcTransport {
		t.Errorf("expected `extend-grpc-transport` to be set on the service.")
	}
}

func TestServiceAnnotationsDetailedTracing(t *testing.T) {
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"detailed-tracing-attributes": "true",
	})
	annotateModel(model, codec)
	got := service.Codec.(*serviceAnnotations)
	if !got.DetailedTracingAttributes {
		t.Errorf("serviceAnnotations.DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, true)
	}
}

func TestServiceAnnotationsHasVeneer(t *testing.T) {
	for _, test := range []struct {
		name         string
		hasVeneer    string
		wantService0 bool
		wantService1 bool
	}{
		{
			name:         "all services via true",
			hasVeneer:    "true",
			wantService0: true,
			wantService1: true,
		},
		{
			name:         "false",
			hasVeneer:    "false",
			wantService0: false,
			wantService1: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := newTestAnnotateModelAPI(t)
			service0 := model.Service("..Service0")
			if service0 == nil {
				t.Fatal("cannot find ..Service0")
			}
			service1 := model.Service("..Service1")
			if service1 == nil {
				t.Fatal("cannot find ..Service1")
			}

			options := map[string]string{}
			if test.hasVeneer != "" {
				options["has-veneer"] = test.hasVeneer
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", options)
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}

			ann0 := service0.Codec.(*serviceAnnotations)
			if ann0.HasVeneer != test.wantService0 {
				t.Errorf("Service0 HasVeneer = %v, want %v", ann0.HasVeneer, test.wantService0)
			}
			for _, m := range ann0.Methods {
				if mAnn := m.Codec.(*methodAnnotation); mAnn.HasVeneer != test.wantService0 {
					t.Errorf("method %s HasVeneer = %v, want %v", m.Name, mAnn.HasVeneer, test.wantService0)
				}
			}

			ann1 := service1.Codec.(*serviceAnnotations)
			if ann1.HasVeneer != test.wantService1 {
				t.Errorf("Service1 HasVeneer = %v, want %v", ann1.HasVeneer, test.wantService1)
			}
			for _, m := range ann1.Methods {
				if mAnn := m.Codec.(*methodAnnotation); mAnn.HasVeneer != test.wantService1 {
					t.Errorf("method %s HasVeneer = %v, want %v", m.Name, mAnn.HasVeneer, test.wantService1)
				}
			}
		})
	}
}

func TestServiceAnnotationsPerServiceFeatures(t *testing.T) {
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"per-service-features": "true",
	})
	annotateModel(model, codec)
	wantService := &serviceAnnotations{
		Name:               "ResourceService",
		PackageModuleName:  "test::v1",
		ModuleName:         "resource_service",
		PerServiceFeatures: true,
		Incomplete:         true,
		GrpcClient:         "gaxi::grpc::Client",
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceAnnotationsAPIVersions(t *testing.T) {
	setSingleMethodVersion := func(t *testing.T, model *api.API) {
		t.Helper()
		id := ".test.v1.ResourceService.GetResource"
		method := model.Method(id)
		if method == nil {
			t.Fatalf("cannot find method %s", id)
		}
		method.WithAPIVersion("v1_20260205")
	}
	setMultipleMethodVersions := func(t *testing.T, model *api.API) {
		t.Helper()
		setSingleMethodVersion(t, model)
		id := ".test.v1.ResourceService.DeleteResource"
		method := model.Method(id)
		if method == nil {
			t.Fatalf("cannot find method %s", id)
		}
		method.WithAPIVersion("v1_20270305")
	}

	for _, test := range []struct {
		wantVersion string
		delta       func(t *testing.T, model *api.API)
	}{
		{
			wantVersion: "",
			delta:       func(_ *testing.T, _ *api.API) {},
		},
		{
			wantVersion: "",
			delta: func(t *testing.T, model *api.API) {
				id := ".test.v1.ResourceService"
				service := model.Service(id)
				if service == nil {
					t.Fatalf("cannot find service %s", id)
				}
				service.Methods = []*api.Method{}
			},
		},
		{
			wantVersion: "v1_20260205",
			delta:       setSingleMethodVersion,
		},
		{
			wantVersion: "v1_20270305",
			delta:       setMultipleMethodVersions,
		},
	} {
		t.Run(test.wantVersion, func(t *testing.T) {
			model := serviceAnnotationsModel()
			test.delta(t, model)
			id := ".test.v1.ResourceService"
			service := model.Service(id)
			if service == nil {
				t.Fatalf("cannot find service %s", id)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}
			got := service.Codec.(*serviceAnnotations)
			if got == nil {
				t.Fatalf("no annotations for service %s", service.ID)
			}
			if gotVersion := got.MaximumAPIVersion(); gotVersion != test.wantVersion {
				t.Errorf("got.MaximumAPIVersion() = %q, want = %q", gotVersion, test.wantVersion)
			}
		})
	}
}

func TestServiceAnnotationsLROTypes(t *testing.T) {
	createReq := api.NewTestMessage("CreateResourceRequest")
	deleteReq := api.NewTestMessage("DeleteResourceRequest")
	resource := api.NewTestMessage("Resource")
	metadata := api.NewTestMessage("OperationMetadata")
	operation := api.NewTestMessage("Operation").WithPackage("google.longrunning")

	methodCreate := api.NewTestMethod("CreateResource").
		WithInput(createReq).
		WithOutput(operation).
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: metadata.ID,
			ResponseTypeID: resource.ID,
		}).
		WithBindings()

	methodDelete := api.NewTestMethod("DeleteResource").
		WithInput(deleteReq).
		WithOutput(operation).
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: metadata.ID,
			ResponseTypeID: api.WktEmptyID,
		}).
		WithBindings()

	service := api.NewTestService("LroService").
		WithMethods(methodCreate, methodDelete)

	model := api.NewTestAPI([]*api.Message{createReq, deleteReq, resource, metadata, operation}, nil, []*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{
		"include-grpc-only-methods": "true",
	})
	codec.packageMapping["google.longrunning"] = &packagez{name: "google-cloud-longrunning"}
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}
	empty := model.Message(api.WktEmptyID)
	wantService := &serviceAnnotations{
		Name:              "LroService",
		PackageModuleName: "test",
		ModuleName:        "lro_service",
		LROTypes: []*api.Message{
			metadata,
			resource,
			empty,
		},
		GrpcClient: "gaxi::grpc::Client",
	}
	if !wantService.HasLROs() {
		t.Errorf("HasLRO should be true. The service has several LROs.")
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceAnnotationsNameOverrides(t *testing.T) {
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	method := model.Method(".test.v1.ResourceService.GetResource")
	if method == nil {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"name-overrides": ".test.v1.ResourceService=Renamed",
	})
	annotateModel(model, codec)

	serviceFilter := cmpopts.IgnoreFields(serviceAnnotations{}, "PackageModuleName", "Methods")
	wantService := &serviceAnnotations{
		Name:       "Renamed",
		ModuleName: "renamed",
		Incomplete: true,
		GrpcClient: "gaxi::grpc::Client",
	}
	if diff := cmp.Diff(wantService, service.Codec, serviceFilter); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	methodFilter := cmpopts.IgnoreFields(methodAnnotation{}, "Name", "NameNoMangling", "BuilderName", "Body", "PathInfo", "SystemParameters", "ReturnType", "IsGrpc")
	wantMethod := &methodAnnotation{
		ServiceNameToPascal: "Renamed",
		ServiceNameToCamel:  "renamed",
		ServiceNameToSnake:  "renamed",
	}
	if diff := cmp.Diff(wantMethod, method.Codec, methodFilter); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceAnnotations(t *testing.T) {
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	method := model.Method(".test.v1.ResourceService.GetResource")
	if method == nil {
		t.Fatal("cannot find .test.v1.ResourceService.GetResource")
	}
	emptyMethod := model.Method(".test.v1.ResourceService.DeleteResource")
	if emptyMethod == nil {
		t.Fatal("cannot find .test.v1.ResourceService.DeleteResource")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{})
	annotateModel(model, codec)
	wantService := &serviceAnnotations{
		Name:              "ResourceService",
		PackageModuleName: "test::v1",
		ModuleName:        "resource_service",
		Incomplete:        true,
		GrpcClient:        "gaxi::grpc::Client",
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// The `noHttpMethod` should be excluded from the list of methods in the
	// Codec.
	serviceAnn := service.Codec.(*serviceAnnotations)
	wantMethodList := []*api.Method{method, emptyMethod}
	if diff := cmp.Diff(wantMethodList, serviceAnn.Methods, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{}), cmpopts.IgnoreFields(api.Method{}, "Model", "Service", "SourceService")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethod := &methodAnnotation{
		Name:           "get_resource",
		NameNoMangling: "get_resource",
		BuilderName:    "GetResource",
		Body:           "None::<gaxi::http::NoBody>",
		PathInfo:       method.PathInfo,
		SystemParameters: []systemParameter{
			{Name: "$alt", Value: "json;enum-encoding=int"},
		},
		ServiceNameToPascal: "ResourceService",
		ServiceNameToCamel:  "resourceService",
		ServiceNameToSnake:  "resource_service",
		ReturnType:          "crate::model::Response",
	}
	if diff := cmp.Diff(wantMethod, method.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantMethod = &methodAnnotation{
		Name:           "delete_resource",
		NameNoMangling: "delete_resource",
		BuilderName:    "DeleteResource",
		Body:           "None::<gaxi::http::NoBody>",
		PathInfo:       emptyMethod.PathInfo,
		SystemParameters: []systemParameter{
			{Name: "$alt", Value: "json;enum-encoding=int"},
		},
		ServiceNameToPascal: "ResourceService",
		ServiceNameToCamel:  "resourceService",
		ServiceNameToSnake:  "resource_service",
		ReturnType:          "()",
	}
	if diff := cmp.Diff(wantMethod, emptyMethod.Codec, cmpopts.IgnoreUnexported(pathBindingAnnotation{}, bindingSubstitution{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceAnnotationsStreaming(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")

	bidiChat := api.NewTestMethod("Chat").
		WithInput(msg).
		WithOutput(msg).
		WithBidiStreaming().
		WithBindings()
	bidiUnary := api.NewTestMethod("Unary").
		WithInput(msg).
		WithOutput(msg).
		WithBindings()
	bidiService := api.NewTestService("BidiService").
		WithPackage("test.v1").
		WithMethods(bidiChat, bidiUnary)

	serverExpand := api.NewTestMethod("Expand").
		WithInput(msg).
		WithOutput(msg).
		WithServerSideStreaming().
		WithBindings()
	serverUnary := api.NewTestMethod("Unary").
		WithInput(msg).
		WithOutput(msg).
		WithBindings()
	serverService := api.NewTestService("ServerService").
		WithPackage("test.v1").
		WithMethods(serverExpand, serverUnary)

	unaryGet := api.NewTestMethod("Get").
		WithInput(msg).
		WithOutput(msg).
		WithBindings()
	unaryService := api.NewTestService("UnaryService").
		WithPackage("test.v1").
		WithMethods(unaryGet)

	for _, test := range []struct {
		name          string
		service       *api.Service
		options       map[string]string
		wantBidi      bool
		wantServer    bool
		wantStreaming bool
	}{
		{
			name:          "bidi service",
			service:       bidiService,
			options:       map[string]string{},
			wantBidi:      true,
			wantServer:    false,
			wantStreaming: true,
		},
		{
			name:          "server streaming service",
			service:       serverService,
			options:       map[string]string{},
			wantBidi:      false,
			wantServer:    true,
			wantStreaming: true,
		},
		{
			name:          "non-streaming service",
			service:       unaryService,
			options:       map[string]string{},
			wantBidi:      false,
			wantServer:    false,
			wantStreaming: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{test.service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}
			serviceAnn := test.service.Codec.(*serviceAnnotations)
			if got := serviceAnn.HasBidiStreaming(); got != test.wantBidi {
				t.Errorf("serviceAnnotations.HasBidiStreaming() = %v, want %v", got, test.wantBidi)
			}
			if got := serviceAnn.HasServerStreaming(); got != test.wantServer {
				t.Errorf("serviceAnnotations.HasServerStreaming() = %v, want %v", got, test.wantServer)
			}
			if got := serviceAnn.HasStreaming(); got != test.wantStreaming {
				t.Errorf("serviceAnnotations.HasStreaming() = %v, want %v", got, test.wantStreaming)
			}
		})
	}
}

func TestServiceAnnotationsMethodKinds(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")
	bidiMethod := api.NewTestMethod("Chat").WithInput(msg).WithOutput(msg).WithBidiStreaming().WithPathTemplate(&api.PathTemplate{})
	serverMethod := api.NewTestMethod("Expand").WithInput(msg).WithOutput(msg).WithServerSideStreaming().WithPathTemplate(&api.PathTemplate{})
	unaryMethod := api.NewTestMethod("Get").WithInput(msg).WithOutput(msg).WithVerb("GET").WithPathTemplate(&api.PathTemplate{})

	for _, test := range []struct {
		name                  string
		methods               []*api.Method
		options               map[string]string
		wantRequestBuilder    bool
		wantBidiStreamBuilder bool
	}{
		{
			name:                  "bidi only",
			methods:               []*api.Method{bidiMethod},
			options:               map[string]string{},
			wantRequestBuilder:    false,
			wantBidiStreamBuilder: true,
		},
		{
			name:                  "bidi and unary",
			methods:               []*api.Method{bidiMethod, unaryMethod},
			options:               map[string]string{},
			wantRequestBuilder:    true,
			wantBidiStreamBuilder: true,
		},
		{
			name:                  "server streaming only",
			methods:               []*api.Method{serverMethod},
			options:               map[string]string{},
			wantRequestBuilder:    true,
			wantBidiStreamBuilder: false,
		},
		{
			name:                  "server streaming and bidi",
			methods:               []*api.Method{serverMethod, bidiMethod},
			options:               map[string]string{},
			wantRequestBuilder:    true,
			wantBidiStreamBuilder: true,
		},
		{
			name:                  "unary only",
			methods:               []*api.Method{unaryMethod},
			options:               map[string]string{},
			wantRequestBuilder:    true,
			wantBidiStreamBuilder: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(test.methods...)
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}
			serviceAnn := service.Codec.(*serviceAnnotations)
			if got := serviceAnn.NeedsRequestBuilder(); got != test.wantRequestBuilder {
				t.Errorf("serviceAnnotations.NeedsRequestBuilder() = %v, want %v", got, test.wantRequestBuilder)
			}
			if got := serviceAnn.NeedsBidiStreamBuilder(); got != test.wantBidiStreamBuilder {
				t.Errorf("serviceAnnotations.NeedsBidiStreamBuilder() = %v, want %v", got, test.wantBidiStreamBuilder)
			}
		})
	}
}

func TestServiceAnnotationsClassification(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")
	bidiMethod := api.NewTestMethod("Chat").WithInput(msg).WithOutput(msg).WithBidiStreaming().WithPathTemplate(&api.PathTemplate{})
	unaryMethod := api.NewTestMethod("Get").WithInput(msg).WithOutput(msg).WithVerb("GET").WithPathTemplate(&api.PathTemplate{})

	for _, test := range []struct {
		name       string
		methods    []*api.Method
		options    map[string]string
		isPureGrpc bool
		isPureHttp bool
		isHybrid   bool
	}{
		{
			name:       "pure gRPC service with only streaming methods",
			methods:    []*api.Method{bidiMethod},
			options:    map[string]string{},
			isPureGrpc: true,
			isPureHttp: false,
			isHybrid:   false,
		},
		{
			name:       "pure HTTP service with only unary methods",
			methods:    []*api.Method{unaryMethod},
			options:    map[string]string{},
			isPureGrpc: false,
			isPureHttp: true,
			isHybrid:   false,
		},
		{
			name:       "hybrid service with unary and streaming methods",
			methods:    []*api.Method{unaryMethod, bidiMethod},
			options:    map[string]string{},
			isPureGrpc: false,
			isPureHttp: false,
			isHybrid:   true,
		},
		{
			name:    "default transport grpc marks unary service as pure gRPC",
			methods: []*api.Method{unaryMethod},
			options: map[string]string{
				"default-transport": "grpc",
			},
			isPureGrpc: true,
			isPureHttp: false,
			isHybrid:   false,
		},
		{
			name:    "default transport grpc with streaming is pure gRPC",
			methods: []*api.Method{unaryMethod, bidiMethod},
			options: map[string]string{
				"default-transport": "grpc",
			},
			isPureGrpc: true,
			isPureHttp: false,
			isHybrid:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := api.NewTestService("TestService").WithPackage("test.v1").WithMethods(test.methods...)
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}
			serviceAnn := service.Codec.(*serviceAnnotations)
			if got := serviceAnn.IsPureGrpc(); got != test.isPureGrpc {
				t.Errorf("serviceAnnotations.IsPureGrpc() = %v, want %v", got, test.isPureGrpc)
			}
			if got := serviceAnn.IsPureHttp(); got != test.isPureHttp {
				t.Errorf("serviceAnnotations.IsPureHttp() = %v, want %v", got, test.isPureHttp)
			}
			if got := serviceAnn.IsHybrid(); got != test.isHybrid {
				t.Errorf("serviceAnnotations.IsHybrid() = %v, want %v", got, test.isHybrid)
			}
		})
	}
}
