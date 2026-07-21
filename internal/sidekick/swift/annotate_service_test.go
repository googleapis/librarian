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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService(t *testing.T) {
	for _, test := range []struct {
		name            string
		serviceName     string
		doc             string
		wantAnnotations *serviceAnnotations
		wantImports     []string
	}{
		{
			name:        "IAM service",
			serviceName: "IAM",
			doc:         "IAM service documentation.",
			wantAnnotations: &serviceAnnotations{
				Name:       "IAM",
				ClientName: "IAMClient",
				StubPrefix: "IAM",
				DocLines:   []string{"IAM service documentation."},
			},
			wantImports: []string{"GoogleCloudWkt"},
		},
		{
			name:        "Service with mangled name",
			serviceName: "Protocol",
			doc:         "Docs are not relevant.",
			wantAnnotations: &serviceAnnotations{
				Name:       "Protocol_",
				ClientName: "ProtocolClient",
				StubPrefix: "Protocol",
				DocLines:   []string{"Docs are not relevant."},
			},
			wantImports: []string{"GoogleCloudWkt"},
		},
		{
			name:        "SecretManagerService",
			serviceName: "SecretManagerService",
			doc:         "Secret Manager Service documentation.\nLine 2.",
			wantAnnotations: &serviceAnnotations{
				Name:       "SecretManagerService",
				ClientName: "SecretManagerServiceClient",
				StubPrefix: "SecretManagerService",
				DocLines:   []string{"Secret Manager Service documentation.", "Line 2."},
			},
			wantImports: []string{"GoogleCloudWkt"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &api.Service{
				Name:          test.serviceName,
				Documentation: test.doc,
			}
			model := api.NewTestAPI(nil, nil, []*api.Service{s})
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantAnnotations, s.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "PackageName", "QuickstartMethod", "Model", "DependsOn")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			annotations := s.Codec.(*serviceAnnotations)
			if diff := cmp.Diff(test.wantImports, annotations.ServiceImports()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateService_SkipNoBindings(t *testing.T) {
	inputType := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
	}
	outputType := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	service := &api.Service{
		Name:    "TestService",
		ID:      ".test.TestService",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "ValidMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
			{
				Name:         "NoBindingMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{},
				},
			},
			{
				Name:         "NilPathInfoMethod",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
			},
		},
	}

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	serviceCodec := service.Codec.(*serviceAnnotations)
	var gotNames []string
	for _, m := range serviceCodec.RestMethods {
		gotNames = append(gotNames, m.Name)
	}
	wantNames := []string{"ValidMethod"}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_Quickstart(t *testing.T) {
	for _, test := range []struct {
		name             string
		quickstartMethod *api.Method
		wantQuickstart   bool
	}{
		{
			name:             "nil quickstart",
			quickstartMethod: nil,
			wantQuickstart:   false,
		},
		{
			name: "non-generated quickstart (nil PathInfo)",
			quickstartMethod: &api.Method{
				Name:     "Quickstart",
				PathInfo: nil,
			},
			wantQuickstart: false,
		},
		{
			name: "non-generated quickstart (empty bindings)",
			quickstartMethod: &api.Method{
				Name: "Quickstart",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{},
				},
			},
			wantQuickstart: false,
		},
		{
			name: "generated quickstart",
			quickstartMethod: &api.Method{
				Name: "Quickstart",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
			wantQuickstart: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &api.Service{
				Name:             "TestService",
				QuickstartMethod: test.quickstartMethod,
			}

			model := api.NewTestAPI(nil, nil, []*api.Service{service})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			annotations, ok := service.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatal("service.Codec is not *serviceAnnotations")
			}

			if test.wantQuickstart {
				if annotations.QuickstartMethod == nil {
					t.Error("expected QuickstartMethod to be set, got nil")
				}
			} else {
				if annotations.QuickstartMethod != nil {
					t.Errorf("expected QuickstartMethod to be nil, got %v", annotations.QuickstartMethod)
				}
			}
		})
	}
}

func TestAnnotateService_Gating(t *testing.T) {
	model := makeGatedTestModel()
	codec := newTestCodec(t, model, nil)
	codec.PerServiceTraits = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, service := range model.Services {
		t.Run(service.Name, func(t *testing.T) {
			ann, ok := service.Codec.(*serviceAnnotations)
			if !ok {
				t.Fatalf("expected service.Codec to be *serviceAnnotations, got %T", service.Codec)
			}
			if !ann.IsGated {
				t.Error("expected IsGated to be true when PerServiceTraits is true")
			}
		})
	}
}

func TestAnnotateService_RequiredServices(t *testing.T) {
	model := makeRequiredServicesTestModel()
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "external", Name: "GoogleCloudExternal"},
	})
	codec.PerServiceTraits = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	targetService := model.Service(".test.TestService")
	if targetService == nil {
		t.Fatalf("missing target service .test.zoneOperations")
	}
	targetCodec, ok := targetService.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("expected targetService.Codec to be *serviceAnnotations, got %T", targetService.Codec)
	}

	sourceService := model.Service(".test.zoneOperations")
	if sourceService == nil {
		t.Fatalf("missing source service .test.zoneOperations")
	}
	wantRequired := map[string]*api.Service{
		sourceService.ID: sourceService,
	}
	if diff := cmp.Diff(wantRequired, targetCodec.RequiredServices, cmpopts.IgnoreFields(api.Method{}, "Model"), cmpopts.IgnoreFields(api.Service{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_LRO(t *testing.T) {
	inputType := &api.Message{
		Name:    "Request",
		Package: "test",
		ID:      ".test.Request",
	}
	outputType := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
	}
	lroResponseType := &api.Message{
		Name:    "LroResponse",
		Package: "external",
		ID:      ".external.LroResponse",
	}
	lroMetadataType := &api.Message{
		Name:    "LroMetadata",
		Package: "external",
		ID:      ".external.LroMetadata",
	}

	method := &api.Method{
		Name:         "LroMethod",
		InputTypeID:  inputType.ID,
		InputType:    inputType,
		OutputTypeID: outputType.ID,
		OutputType:   outputType,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
		},
		IsLRO: true,
		OperationInfo: &api.OperationInfo{
			ResponseTypeID: lroResponseType.ID,
			MetadataTypeID: lroMetadataType.ID,
		},
	}

	service := &api.Service{
		Name:    "TestService",
		ID:      ".test.TestService",
		Package: "test",
		Methods: []*api.Method{method},
	}

	model := api.NewTestAPI([]*api.Message{inputType, outputType, lroResponseType, lroMetadataType}, nil, []*api.Service{service})
	model.PackageName = "test"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "google.rpc",
			Name:       "GoogleRpc",
		},
		{
			ApiPackage: "external",
			Name:       "GoogleCloudExternal",
		},
		{
			ApiPackage: "google.longrunning",
			Name:       "GoogleLongrunning",
		},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	annotations := service.Codec.(*serviceAnnotations)
	wantImports := []string{"GoogleCloudExternal", "GoogleCloudWkt", "GoogleLongrunning", "GoogleRpc"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_Pagination(t *testing.T) {
	itemType := api.NewTestMessage("Item").WithPackage("external")

	pageToken := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("ListItemsRequest").
		WithFields(api.NewTestField("name").WithType(api.TypezString)).
		WithFields(pageToken)
	outputType := api.NewTestMessage("ListItemsResponse").
		WithPagination(
			api.NewTestField("next_page_token").WithType(api.TypezString),
			api.NewTestField("items").WithMessageType(itemType).WithRepeated(),
		)
	list := api.NewTestMethod("ListItems").
		WithInput(inputType).
		WithOutput(outputType).
		WithPagination(pageToken).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("items"))

	service := api.NewTestService("TestService").WithMethods(list)

	model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
	model.PackageName = "test"
	model.AddMessage(itemType)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "external",
			Name:       "GoogleCloudExternal",
		},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	annotations := service.Codec.(*serviceAnnotations)
	wantImports := []string{"GoogleCloudExternal", "GoogleCloudWkt"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_MapPagination(t *testing.T) {
	itemType := api.NewTestMessage("Item").WithPackage("external")
	mapType := api.NewTestMessage("$map<string, Item>").
		WithPackage("test").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithMessageType(itemType),
		)
	mapType.IsMap = true

	pageToken := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("ListItemsRequest").
		WithFields(api.NewTestField("name").WithType(api.TypezString)).
		WithFields(pageToken)
	outputType := api.NewTestMessage("ListItemsResponse").
		WithPagination(
			api.NewTestField("next_page_token").WithType(api.TypezString),
			api.NewTestField("items").WithMessageType(mapType).WithMap(),
		)
	list := api.NewTestMethod("ListItems").
		WithInput(inputType).
		WithOutput(outputType).
		WithPagination(pageToken).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("items"))

	service := api.NewTestService("TestService").WithMethods(list)

	model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
	model.PackageName = "test"
	model.AddMessage(itemType)
	model.AddMessage(mapType)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{
			ApiPackage: "external",
			Name:       "GoogleCloudExternal",
		},
	})

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	annotations := service.Codec.(*serviceAnnotations)
	wantImports := []string{"GoogleCloudExternal", "GoogleCloudWkt"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_MethodSignatures(t *testing.T) {
	for _, test := range []struct {
		name        string
		signatures  []*api.MethodSignature
		wantImports []string
	}{
		{
			name:        "no signature",
			signatures:  nil,
			wantImports: []string{"GoogleCloudWkt"},
		},
		{
			name:        "unrealistic, but good for testing",
			signatures:  []*api.MethodSignature{{Names: []string{"parent", "thing_id"}}},
			wantImports: []string{"GoogleCloudWkt"},
		},
		{
			name:        "with external field",
			signatures:  []*api.MethodSignature{{Names: []string{"parent", "thing_id", "external_thing"}}},
			wantImports: []string{"GoogleCloudExternal", "GoogleCloudWkt"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			thing := api.NewTestMessage("Thing").WithPackage("external")
			inputType := api.NewTestMessage("Request").
				WithFields(
					api.NewTestField("parent").WithType(api.TypezString),
					api.NewTestField("thing_id").WithType(api.TypezString),
					api.NewTestField("external_thing").WithMessageType(thing),
				)
			outputType := api.NewTestMessage("Response")
			create := api.NewTestMethod("CreateThing").
				WithInput(inputType).
				WithOutput(outputType).
				WithSignatures(test.signatures...).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things"))
			service := api.NewTestService("TestService").WithMethods(create)
			model := api.NewTestAPI([]*api.Message{inputType, outputType}, nil, []*api.Service{service})
			model.PackageName = "test"
			model.AddMessage(thing)
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, model, nil)
			codec.withExtraDependencies(t, []config.SwiftDependency{
				{
					ApiPackage: "external",
					Name:       "GoogleCloudExternal",
				},
			})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			annotations := service.Codec.(*serviceAnnotations)
			if annotations == nil {
				t.Fatalf("service should have a `serviceAnnotations`, got=%+v", service.Codec)
			}
			if diff := cmp.Diff(test.wantImports, annotations.ServiceImports()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
