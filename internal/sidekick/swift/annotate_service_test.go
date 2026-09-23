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
				Name:        "IAM",
				LibraryName: "Test",
				ClientName:  "IAMClient",
				StubPrefix:  "IAM",
				DocLines:    []string{"IAM service documentation."},
			},
			wantImports: []string{},
		},
		{
			name:        "Service with mangled name",
			serviceName: "Protocol",
			doc:         "Docs are not relevant.",
			wantAnnotations: &serviceAnnotations{
				Name:        "Protocol_",
				LibraryName: "Test",
				ClientName:  "ProtocolClient",
				StubPrefix:  "Protocol",
				DocLines:    []string{"Docs are not relevant."},
			},
			wantImports: []string{},
		},
		{
			name:        "SecretManagerService",
			serviceName: "SecretManagerService",
			doc:         "Secret Manager Service documentation.\nLine 2.",
			wantAnnotations: &serviceAnnotations{
				Name:        "SecretManagerService",
				LibraryName: "Test",
				ClientName:  "SecretManagerServiceClient",
				StubPrefix:  "SecretManagerService",
				DocLines:    []string{"Secret Manager Service documentation.", "Line 2."},
			},
			wantImports: []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := api.NewTestService(test.serviceName)
			s.Documentation = test.doc
			model := api.NewTestAPI(nil, nil, []*api.Service{s})
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantAnnotations, s.Codec, cmpopts.IgnoreFields(serviceAnnotations{}, "QuickstartMethod", "Model", "DependsOn", "PublicDependsOn")); diff != "" {
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
	inputType := api.NewTestMessage("Request")
	outputType := api.NewTestMessage("Response")
	validMethod := api.NewTestMethod("ValidMethod").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(&api.PathTemplate{})
	noBindingMethod := api.NewTestMethod("NoBindingMethod").
		WithInput(inputType).
		WithOutput(outputType).
		WithBindings()
	nilPathInfoMethod := api.NewTestMethod("NilPathInfoMethod").
		WithInput(inputType).
		WithOutput(outputType)
	nilPathInfoMethod.PathInfo = nil

	service := api.NewTestService("TestService").
		WithMethods(validMethod, noBindingMethod, nilPathInfoMethod)

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	serviceCodec, ok := service.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("mismatched service annotations, got=%T", service.Codec)
	}
	if serviceCodec.HasLROs() {
		t.Errorf("expected HasLROs() == false, annotations=%+v", serviceCodec)
	}
	var gotNames []string
	for _, m := range serviceCodec.Methods {
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
			quickstartMethod: func() *api.Method {
				m := api.NewTestMethod("Quickstart")
				m.PathInfo = nil
				return m
			}(),
			wantQuickstart: false,
		},
		{
			name:             "non-generated quickstart (empty bindings)",
			quickstartMethod: api.NewTestMethod("Quickstart").WithBindings(),
			wantQuickstart:   false,
		},
		{
			name: "generated quickstart",
			quickstartMethod: api.NewTestMethod("Quickstart").
				WithVerb("GET").
				WithPathTemplate(&api.PathTemplate{}),
			wantQuickstart: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := api.NewTestService("TestService")
			service.QuickstartMethod = test.quickstartMethod

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
	inputType := api.NewTestMessage("Request")
	outputType := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	lroResponseType := api.NewTestMessage("LroResponse").WithPackage("external")
	lroMetadataType := api.NewTestMessage("LroMetadata").WithPackage("external")

	method := api.NewTestMethod("LroMethod").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("POST").
		WithPathTemplate(&api.PathTemplate{}).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: lroResponseType.ID,
			MetadataTypeID: lroMetadataType.ID,
		})

	service := api.NewTestService("TestService").WithMethods(method)

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

	annotations, ok := service.Codec.(*serviceAnnotations)
	if !ok {
		t.Fatalf("expected `serviceAnnotations`, got %T", service.Codec)
	}
	if !annotations.HasLROs() {
		t.Errorf("expected HasLROs() == true, annotations=%+v", annotations)
	}
	wantImports := []string{"GoogleCloudExternal", "GoogleLongrunning", "GoogleRpc"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantPublicImports := []string{"GoogleCloudExternal", "GoogleLongrunning"}
	if diff := cmp.Diff(wantPublicImports, annotations.PublicServiceImports()); diff != "" {
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
	wantImports := []string{"GoogleCloudExternal"}
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
	wantImports := []string{"GoogleCloudExternal"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_MethodSignatures(t *testing.T) {
	for _, test := range []struct {
		name        string
		signatures  []*api.MethodSignature
		wantImports []string
		wantHasData bool
	}{
		{
			name:        "no signature",
			signatures:  nil,
			wantImports: []string{},
		},
		{
			name:        "unrealistic, but good for testing",
			signatures:  []*api.MethodSignature{{Names: []string{"parent", "thing_id"}}},
			wantImports: []string{},
		},
		{
			name:        "with external field",
			signatures:  []*api.MethodSignature{{Names: []string{"parent", "thing_id", "external_thing"}}},
			wantImports: []string{"GoogleCloudExternal"},
		},
		{
			name:        "with bytes field",
			signatures:  []*api.MethodSignature{{Names: []string{"parent", "data"}}},
			wantImports: []string{},
			wantHasData: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			thing := api.NewTestMessage("Thing").WithPackage("external")
			inputType := api.NewTestMessage("Request").
				WithFields(
					api.NewTestField("parent").WithType(api.TypezString),
					api.NewTestField("thing_id").WithType(api.TypezString),
					api.NewTestField("external_thing").WithMessageType(thing),
					api.NewTestField("data").WithType(api.TypezBytes),
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
			if annotations.HasData != test.wantHasData {
				t.Errorf("HasData mismatch: got %v, want %v", annotations.HasData, test.wantHasData)
			}
		})
	}
}

func TestAnnotateService_WktImports(t *testing.T) {
	inputType := api.NewTestMessage("DeleteThingRequest").WithPackage("test")
	method := api.NewTestMethod("DeleteThing").
		WithInput(inputType).
		WithVerb("DELETE").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things"))
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{inputType}, nil, []*api.Service{service})
	model.PackageName = "test"
	wktEmpty := model.Message(".google.protobuf.Empty")
	if wktEmpty == nil {
		t.Fatal("expected .google.protobuf.Empty in model")
	}
	method.WithOutput(wktEmpty)
	method.ReturnsEmpty = true
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	annotations := service.Codec.(*serviceAnnotations)
	wantImports := []string{"GoogleWKT"}
	if diff := cmp.Diff(wantImports, annotations.ServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantPublicImports := []string{}
	if diff := cmp.Diff(wantPublicImports, annotations.PublicServiceImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateService_MapFieldDependencies(t *testing.T) {
	keyField := api.NewTestField("key").WithType(api.TypezString)
	valField := api.NewTestField("value").WithType(api.TypezBytes)
	mapEntry := api.NewTestMessage("DataMapEntry").
		WithPackage("test").
		WithIsMap().
		WithFields(keyField, valField)

	mapField := api.NewTestField("data_map").
		WithMap().
		WithMessageType(mapEntry)

	signature := &api.MethodSignature{
		Fields: []*api.Field{mapField},
	}
	inputType := api.NewTestMessage("Request").
		WithPackage("test").
		WithFields(mapField)
	outputType := api.NewTestMessage("Response").WithPackage("test")

	method := api.NewTestMethod("UploadMap").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("POST").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("upload")).
		WithSignatures(signature)

	service := api.NewTestService("MapService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{inputType, outputType, mapEntry}, nil, []*api.Service{service})
	model.PackageName = "test"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	annotations := service.Codec.(*serviceAnnotations)
	if !annotations.HasData {
		t.Errorf("expected annotations.HasData to be true for signature with map<string, bytes>")
	}
}

func TestAnnotateService_SnippetImports(t *testing.T) {
	for _, test := range []struct {
		name               string
		updateMask         bool
		externalDep        bool
		wantSnippetImports []string
	}{
		{
			name:               "no update mask and no external dep",
			updateMask:         false,
			externalDep:        false,
			wantSnippetImports: nil,
		},
		{
			name:               "with update mask",
			updateMask:         true,
			externalDep:        false,
			wantSnippetImports: []string{"GoogleWKT"},
		},
		{
			name:               "with external dep and no update mask",
			updateMask:         false,
			externalDep:        true,
			wantSnippetImports: []string{"GoogleCloudExternal"},
		},
		{
			name:               "with update mask and external dep",
			updateMask:         true,
			externalDep:        true,
			wantSnippetImports: []string{"GoogleCloudExternal", "GoogleWKT"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputType := api.NewTestMessage("Request").WithPackage("test")
			outputType := api.NewTestMessage("Response").WithPackage("test")
			method := api.NewTestMethod("DoThing").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things"))

			var messages []*api.Message
			messages = append(messages, inputType, outputType)

			if test.externalDep {
				extType := api.NewTestMessage("External").WithPackage("google.cloud.external")
				messages = append(messages, extType)
				method.WithOutput(extType)
			}

			service := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI(messages, nil, []*api.Service{service})
			model.PackageName = "test"
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			if test.updateMask {
				maskField := api.NewTestField("update_mask")
				method.WithSampleInfo(&api.SampleInfo{UpdateMaskField: maskField})
			}

			codec := newTestCodec(t, model, nil)
			if test.externalDep {
				codec.withExtraDependencies(t, []config.SwiftDependency{
					{Name: "GoogleCloudExternal", ApiPackage: "google.cloud.external"},
				})
			}
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			annotations := service.Codec.(*serviceAnnotations)
			if diff := cmp.Diff(test.wantSnippetImports, annotations.SnippetImports(), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("SnippetImports mismatch (-want +got):\n%s", diff)
			}
			if test.updateMask != annotations.needsWktImport() {
				t.Errorf("needsWktImport mismatch: got %v, want %v", annotations.needsWktImport(), test.updateMask)
			}
		})
	}
}
