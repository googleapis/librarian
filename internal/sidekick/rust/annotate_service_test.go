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
	model := serviceAnnotationsModel()
	service := model.Service(".test.v1.ResourceService")
	if service == nil {
		t.Fatal("cannot find .test.v1.ResourceService")
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"has-veneer": "true",
	})
	annotateModel(model, codec)
	serviceAnn := service.Codec.(*serviceAnnotations)

	if !serviceAnn.HasVeneer {
		t.Errorf("expected `has-veneer` to be set on the service.")
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
		method.APIVersion = "v1_20260205"
	}
	setMultipleMethodVersions := func(t *testing.T, model *api.API) {
		t.Helper()
		setSingleMethodVersion(t, model)
		id := ".test.v1.ResourceService.DeleteResource"
		method := model.Method(id)
		if method == nil {
			t.Fatalf("cannot find method %s", id)
		}
		method.APIVersion = "v1_20270305"
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
	create := &api.Message{
		Name:    "CreateResourceRequest",
		ID:      ".test.CreateResourceRequest",
		Package: "test",
	}
	delete := &api.Message{
		Name:    "DeleteResourceRequest",
		ID:      ".test.DeleteResourceRequest",
		Package: "test",
	}
	resource := &api.Message{
		Name:    "Resource",
		ID:      ".test.Resource",
		Package: "test",
	}
	metadata := &api.Message{
		Name:    "OperationMetadata",
		ID:      ".test.OperationMetadata",
		Package: "test",
	}
	operation := &api.Message{
		Name:    "Operation",
		ID:      ".google.longrunning.Operation",
		Package: "google.longrunning",
	}
	service := &api.Service{
		Name:    "LroService",
		ID:      ".test.LroService",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "CreateResource",
				ID:           ".test.LroService.CreateResource",
				PathInfo:     &api.PathInfo{},
				InputType:    create,
				InputTypeID:  ".test.CreateResourceRequest",
				OutputTypeID: ".google.longrunning.Operation",
				OperationInfo: &api.OperationInfo{
					MetadataTypeID: ".test.OperationMetadata",
					ResponseTypeID: ".test.Resource",
				},
			},
			{
				Name:         "DeleteResource",
				ID:           ".test.LroService.DeleteResource",
				PathInfo:     &api.PathInfo{},
				InputType:    delete,
				InputTypeID:  ".test.DeleteResourceRequest",
				OutputTypeID: ".google.longrunning.Operation",
				OperationInfo: &api.OperationInfo{
					MetadataTypeID: ".test.OperationMetadata",
					ResponseTypeID: ".google.protobuf.Empty",
				},
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{create, delete, resource, metadata, operation}, []*api.Enum{}, []*api.Service{service})
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "test", map[string]string{
		"include-grpc-only-methods": "true",
	})
	codec.packageMapping["google.longrunning"] = &packagez{name: "google-cloud-longrunning"}
	annotateModel(model, codec)
	empty := model.Message(".google.protobuf.Empty")
	wantService := &serviceAnnotations{
		Name:              "LroService",
		PackageModuleName: "test",
		ModuleName:        "lro_service",
		LROTypes: []*api.Message{
			metadata,
			resource,
			empty,
		},
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
	}
	if diff := cmp.Diff(wantService, service.Codec, serviceFilter); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	methodFilter := cmpopts.IgnoreFields(methodAnnotation{}, "Name", "NameNoMangling", "BuilderName", "Body", "PathInfo", "SystemParameters", "ReturnType")
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
	}
	if diff := cmp.Diff(wantService, service.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// The `noHttpMethod` should be excluded from the list of methods in the
	// Codec.
	serviceAnn := service.Codec.(*serviceAnnotations)
	wantMethodList := []*api.Method{method, emptyMethod}
	if diff := cmp.Diff(wantMethodList, serviceAnn.Methods, cmpopts.IgnoreFields(api.Method{}, "Model", "Service", "SourceService")); diff != "" {
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
	if diff := cmp.Diff(wantMethod, method.Codec); diff != "" {
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
	if diff := cmp.Diff(wantMethod, emptyMethod.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
