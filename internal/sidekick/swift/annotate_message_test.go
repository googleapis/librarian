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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	for _, test := range []struct {
		name        string
		message     *api.Message
		want        *messageAnnotations
		wantImports []*dependencyImport
	}{
		{
			name: "simple",
			message: &api.Message{
				Name:          "Secret",
				Documentation: "A secret message.\nWith two lines.",
				ID:            ".test.Secret",
				Package:       "test",
				Fields: []*api.Field{
					{Name: "secret_key", JSONName: "secretKey", Typez: api.TypezString},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				DocLines:            []string{"A secret message.", "With two lines."},
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: false,
				SampleField:         "secretKey",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
		{
			name: "escaped name",
			message: &api.Message{
				Name:          "Protocol",
				Documentation: "A message named Protocol.",
				ID:            ".test.Protocol",
				Package:       "test",
			},
			want: &messageAnnotations{
				Name:                "Protocol_",
				DocLines:            []string{"A message named Protocol."},
				TypeURL:             "type.googleapis.com/test.Protocol",
				CustomSerialization: false,
				SampleField:         "<placeholder>",
				ParameterTypeName:   "Protocol_",
				ProtoTypeName:       "Test_Protocol_",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
		{
			name: "with oneof",
			message: &api.Message{
				Name:    "WithOneof",
				ID:      ".test.WithOneof",
				Package: "test",
				OneOfs:  []*api.OneOf{{Name: "choice"}},
			},
			want: &messageAnnotations{
				Name:                "WithOneof",
				TypeURL:             "type.googleapis.com/test.WithOneof",
				CustomSerialization: true,
				SampleField:         "<placeholder>",
				ParameterTypeName:   "WithOneof",
				ProtoTypeName:       "Test_WithOneof",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
		{
			name: "with custom json name",
			message: &api.Message{
				Name:    "WithCustomJSON",
				ID:      ".test.WithCustomJSON",
				Package: "test",
				Fields: []*api.Field{
					{Name: "secret_key", JSONName: "specialKey", Typez: api.TypezString},
				},
			},
			want: &messageAnnotations{
				Name:                "WithCustomJSON",
				TypeURL:             "type.googleapis.com/test.WithCustomJSON",
				CustomSerialization: true,
				SampleField:         "secretKey",
				ParameterTypeName:   "WithCustomJSON",
				ProtoTypeName:       "Test_WithCustomJSON",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
		{
			name: "with pagination",
			message: &api.Message{
				Name:    "WithPagination",
				ID:      ".test.WithPagination",
				Package: "test",
				Fields: []*api.Field{
					{Name: "secret_key", JSONName: "secretKey", Typez: api.TypezString},
				},
				Pagination: &api.PaginationInfo{
					NextPageToken: &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString},
					PageableItem:  &api.Field{Name: "pageable_item", JSONName: "pageableItem", Typez: api.TypezString, Repeated: true, Codec: &fieldAnnotations{Name: "secretKey", BaseFieldType: "SecretKey"}},
				},
			},
			want: &messageAnnotations{
				Name:                "WithPagination",
				TypeURL:             "type.googleapis.com/test.WithPagination",
				CustomSerialization: false,
				IsPaginatedResponse: true,
				PageableItemField:   "secretKey",
				PageableItemType:    "SecretKey",
				SampleField:         "secretKey",
				ParameterTypeName:   "WithPagination",
				ProtoTypeName:       "Test_WithPagination",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudGax"}, {Module: "GoogleCloudWKT"}},
		},
		{
			name: "service placeholder",
			message: &api.Message{
				Name:               "Service",
				ID:                 ".test.Service",
				Package:            "test",
				ServicePlaceholder: true,
			},
			want: &messageAnnotations{
				Name:                "Service",
				TypeURL:             "type.googleapis.com/test.Service",
				CustomSerialization: false,
				SampleField:         "<placeholder>",
				ParameterTypeName:   "ServiceClient",
				PlaceholderName:     "ServiceClient",
				ProtoTypeName:       "Test_Service",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, f := range test.message.Fields {
				f.Parent = test.message
			}
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantImports, test.message.Codec.(*messageAnnotations).MessageImports()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_ImportAttributes(t *testing.T) {
	noSpiConfig := &config.Library{
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{Name: wellKnownSwiftPackage, ApiPackage: wellKnownProtobufPackage},
					{Name: paginationSwiftPackage, RequiredByServices: true},
				},
			},
		},
	}
	withSpiConfig := &config.Library{
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{Name: wellKnownSwiftPackage, ApiPackage: wellKnownProtobufPackage, SpiAttribute: "Test001"},
					{Name: paginationSwiftPackage, RequiredByServices: true, SpiAttribute: "Test002"},
				},
			},
		},
	}

	for _, test := range []struct {
		name        string
		message     *api.Message
		config      *config.Library
		wantImports []*dependencyImport
	}{
		{
			name:        "simple",
			message:     api.NewTestMessage("Secret"),
			config:      noSpiConfig,
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT"}},
		},
		{
			name:        "with spi attribute",
			message:     api.NewTestMessage("Secret"),
			config:      withSpiConfig,
			wantImports: []*dependencyImport{{Module: "GoogleCloudWKT", Attributes: []string{"@_spi(Test001)"}}},
		},
		{
			name: "with pagination",
			message: api.NewTestMessage("WithPagination").WithPagination(
				api.NewTestField("next_page_token").WithType(api.TypezString),
				api.NewTestField("pageable_item").WithType(api.TypezString).WithRepeated(),
			),
			config:      noSpiConfig,
			wantImports: []*dependencyImport{{Module: "GoogleCloudGax"}, {Module: "GoogleCloudWKT"}},
		},
		{
			name: "with pagination and attributes",
			message: api.NewTestMessage("WithPagination").WithPagination(
				api.NewTestField("next_page_token").WithType(api.TypezString),
				api.NewTestField("pageable_item").WithType(api.TypezString).WithRepeated(),
			),
			config: withSpiConfig,
			wantImports: []*dependencyImport{
				{Module: "GoogleCloudGax", Attributes: []string{"@_spi(Test002)"}},
				{Module: "GoogleCloudWKT", Attributes: []string{"@_spi(Test001)"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, f := range test.message.Fields {
				f.Parent = test.message
			}
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, test.config)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantImports, test.message.Codec.(*messageAnnotations).MessageImports()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_Discovery(t *testing.T) {
	mapMessage := &api.Message{
		Name:  "map<string, bytes>",
		ID:    "$map<string, bytes>",
		IsMap: true,
		Fields: []*api.Field{
			{Name: "key", JSONName: "key", Typez: api.TypezString},
			{Name: "value", JSONName: "value", Typez: api.TypezBytes},
		},
	}

	for _, test := range []struct {
		name    string
		message *api.Message
		want    *messageAnnotations
	}{
		{
			name: "simple",
			message: &api.Message{
				Name:    "Secret",
				ID:      ".test.Secret",
				Package: "test",
				Fields: []*api.Field{
					{Name: "field", JSONName: "field", ID: ".test.Secret.field", Typez: api.TypezString},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: false,
				SampleField:         "field",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
		},
		{
			name: "required",
			message: &api.Message{
				Name:    "Secret",
				ID:      ".test.Secret",
				Package: "test",
				Fields: []*api.Field{
					{Name: "field", JSONName: "field", ID: ".test.Secret.field", Typez: api.TypezBytes},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: true,
				SampleField:         "field",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
		},
		{
			name: "optional",
			message: &api.Message{
				Name:    "Secret",
				ID:      ".test.Secret",
				Package: "test",
				Fields: []*api.Field{
					{Name: "field", JSONName: "field", ID: ".test.Secret.field", Typez: api.TypezBytes, Optional: true},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: true,
				SampleField:         "field",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
		},
		{
			name: "repeated",
			message: &api.Message{
				Name:    "Secret",
				ID:      ".test.Secret",
				Package: "test",
				Fields: []*api.Field{
					{Name: "field", JSONName: "field", ID: ".test.Secret.field", Typez: api.TypezBytes, Repeated: true},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: true,
				SampleField:         "field",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
		},
		{
			name: "map",
			message: &api.Message{
				Name:    "Secret",
				ID:      ".test.Secret",
				Package: "test",
				Fields: []*api.Field{
					{
						Name:     "field",
						JSONName: "field",
						ID:       ".test.Secret.field",
						Typez:    api.TypezMessage,
						TypezID:  mapMessage.ID,
						Map:      true,
					},
				},
			},
			want: &messageAnnotations{
				Name:                "Secret",
				TypeURL:             "type.googleapis.com/test.Secret",
				CustomSerialization: true,
				SampleField:         "field",
				ParameterTypeName:   "Secret",
				ProtoTypeName:       "Test_Secret",
				ModulePath:          "",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, f := range test.message.Fields {
				f.Parent = test.message
			}
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			model.AddMessage(mapMessage)
			codec := newTestCodec(t, model, nil)
			codec.UrlSafeForBytes = true
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_DiscoveryRequests(t *testing.T) {
	for _, test := range []struct {
		name    string
		service *api.Service
		request *api.Message
		want    *messageAnnotations
	}{
		{
			name:    "basic message",
			service: &api.Service{Name: "Service", Package: "test", ID: ".test.Service"},
			request: &api.Message{Name: "getRequest", Package: "test", ID: ".test.Service.getRequest", SyntheticRequest: true},
			want: &messageAnnotations{
				Name:              "GetRequest",
				TypeURL:           "type.googleapis.com/test.Service.getRequest",
				SampleField:       "<placeholder>",
				ParameterTypeName: "ServiceClient.GetRequest",
				ProtoTypeName:     "Test_Service.GetRequest",
				ModulePath:        "",
			},
		},
		{
			name:    "service with reserved name",
			service: &api.Service{Name: "Protocol", Package: "test", ID: ".test.Protocol"},
			request: &api.Message{Name: "listRequest", Package: "test", ID: ".test.Protocol.listRequest", SyntheticRequest: true},
			want: &messageAnnotations{
				Name:              "ListRequest",
				TypeURL:           "type.googleapis.com/test.Protocol.listRequest",
				SampleField:       "<placeholder>",
				ParameterTypeName: "ProtocolClient.ListRequest",
				ProtoTypeName:     "Test_Protocol_.ListRequest",
				ModulePath:        "",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Discovery requests are synthetic. The messages are injected into the data
			// model by sidekick. To avoid clashes, sidekick puts the request messages
			// within a placeholder named after the service.
			servicePlaceholder := &api.Message{
				Name:               test.service.Name,
				Package:            test.service.Package,
				ID:                 test.service.ID,
				ServicePlaceholder: true,
			}
			test.request.Parent = servicePlaceholder
			servicePlaceholder.Messages = append(servicePlaceholder.Messages, test.request)
			model := api.NewTestAPI([]*api.Message{servicePlaceholder}, []*api.Enum{}, []*api.Service{test.service})
			model.AddMessage(test.request)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.request.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_Pagination(t *testing.T) {
	pageSizeField := &api.Field{Name: "page_size", JSONName: "pageSize", Typez: api.TypezInt32}
	pageTokenField := &api.Field{Name: "page_token", JSONName: "pageToken", Typez: api.TypezString}
	inputType := &api.Message{
		Name:    "ListSecretsRequest",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.ListSecretsRequest",
		Fields:  []*api.Field{pageSizeField, pageTokenField},
	}
	pageSizeField.Parent = inputType
	pageTokenField.Parent = inputType

	itemField := &api.Field{Name: "secrets", JSONName: "secrets", Typez: api.TypezMessage, TypezID: ".google.cloud.secretmanager.v1.Secret", Repeated: true}
	nextPageTokenField := &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString}
	outputType := &api.Message{
		Name:    "ListSecretsResponse",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.ListSecretsResponse",
		Fields:  []*api.Field{itemField, nextPageTokenField},
		Pagination: &api.PaginationInfo{
			NextPageToken: nextPageTokenField,
			PageableItem:  itemField,
		},
	}
	itemField.Parent = outputType
	nextPageTokenField.Parent = outputType

	secretType := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.Secret",
	}

	iam := &api.Service{
		Name:    "SecretManagerService",
		ID:      ".google.cloud.secretmanager.v1.SecretManagerService",
		Package: "google.cloud.secretmanager.v1",
		Methods: []*api.Method{
			{
				Name:         "ListSecrets",
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets"),
					}},
				},
				Pagination: pageTokenField,
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.secretmanager.v1"

	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	// Verify annotations on request message
	gotRequest := inputType.Codec.(*messageAnnotations)
	wantRequest := &messageAnnotations{
		Name:              "ListSecretsRequest",
		TypeURL:           "type.googleapis.com/google.cloud.secretmanager.v1.ListSecretsRequest",
		SampleField:       "pageSize",
		ParameterTypeName: "ListSecretsRequest",
		ProtoTypeName:     "Google_Cloud_Secretmanager_V1_ListSecretsRequest",
		ModulePath:        "",
	}
	if diff := cmp.Diff(wantRequest, gotRequest, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantRequestImports := []*dependencyImport{{Module: "GoogleCloudWKT"}}
	if diff := cmp.Diff(wantRequestImports, gotRequest.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify annotations on response message
	gotResponse := outputType.Codec.(*messageAnnotations)
	wantResponse := &messageAnnotations{
		Name:                "ListSecretsResponse",
		TypeURL:             "type.googleapis.com/google.cloud.secretmanager.v1.ListSecretsResponse",
		IsPaginatedResponse: true,
		PageableItemField:   "secrets",
		PageableItemType:    "Secret",
		SampleField:         "secrets",
		ParameterTypeName:   "ListSecretsResponse",
		ProtoTypeName:       "Google_Cloud_Secretmanager_V1_ListSecretsResponse",
		ModulePath:          "",
	}
	if diff := cmp.Diff(wantResponse, gotResponse, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantResponseImports := []*dependencyImport{{Module: "GoogleCloudGax"}, {Module: "GoogleCloudWKT"}}
	if diff := cmp.Diff(wantResponseImports, gotResponse.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMessage_RecursiveNested(t *testing.T) {
	itemField := &api.Field{Name: "secrets", JSONName: "secrets", Typez: api.TypezMessage, TypezID: ".google.cloud.secretmanager.v1.Secret", Repeated: true}
	nextPageTokenField := &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString}
	nestedOutputType := &api.Message{
		Name:    "ListSecretsResponse",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.OuterMessage.ListSecretsResponse",
		Fields:  []*api.Field{itemField, nextPageTokenField},
		Pagination: &api.PaginationInfo{
			NextPageToken: nextPageTokenField,
			PageableItem:  itemField,
		},
	}
	itemField.Parent = nestedOutputType
	nextPageTokenField.Parent = nestedOutputType

	outerMessage := &api.Message{
		Name:     "OuterMessage",
		Package:  "google.cloud.secretmanager.v1",
		ID:       ".google.cloud.secretmanager.v1.OuterMessage",
		Messages: []*api.Message{nestedOutputType},
	}

	secretType := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.Secret",
	}

	model := api.NewTestAPI([]*api.Message{outerMessage, secretType}, nil, nil)
	model.PackageName = "google.cloud.secretmanager.v1"

	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	gotOuter := outerMessage.Codec.(*messageAnnotations)
	wantOuter := &messageAnnotations{
		Name:              "OuterMessage",
		TypeURL:           "type.googleapis.com/google.cloud.secretmanager.v1.OuterMessage",
		SampleField:       "<placeholder>",
		ParameterTypeName: "OuterMessage",
		ProtoTypeName:     "Google_Cloud_Secretmanager_V1_OuterMessage",
		ModulePath:        "",
	}
	if diff := cmp.Diff(wantOuter, gotOuter, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn", "HasConvertedFields")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantImports := []*dependencyImport{{Module: "GoogleCloudGax"}, {Module: "GoogleCloudWKT"}}
	if diff := cmp.Diff(wantImports, gotOuter.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMessage_Gating(t *testing.T) {
	model := makeGatedTestModel()
	codec := newTestCodec(t, model, nil)
	codec.PerServiceTraits = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name           string
		msgName        string
		wantExpression string
	}{
		{"Shared message used by both services", "SharedMessage", "Service1 || Service2"},
		{"Message used by Service1 only", "Service1Message", "Service1"},
		{"Message used by Service2 only", "Service2Message", "Service2"},
		{"Message used by neither service", "UnusedMessage", "Service1 && Service2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := model.Message(fmt.Sprintf(".google.cloud.test.v1.%s", test.msgName))
			if msg == nil {
				t.Fatalf("message %s not found", test.msgName)
			}
			ann, ok := msg.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("expected msg.Codec to be *messageAnnotations, got %T", msg.Codec)
			}

			if diff := cmp.Diff(test.wantExpression, ann.GateExpression()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if !ann.IsGated() {
				t.Error("expected IsGated() to be true")
			}
		})
	}
}

func TestAnnotateMessage_PlaceholderGating(t *testing.T) {
	model := makeRequiredServicesTestModel()
	codec := newTestCodec(t, model, nil)
	codec.withExtraDependencies(t, []config.SwiftDependency{
		{ApiPackage: "external", Name: "GoogleCloudExternal"},
	})
	codec.PerServiceTraits = true

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name  string
		msgID string
		want  string
	}{
		{"placeholder", ".test.zoneOperations", "ZoneOperations"},
		{"operation", ".test.Operation", "TestService || ZoneOperations"},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := model.Message(test.msgID)
			if msg == nil {
				t.Fatalf("message %s not found", test.msgID)
			}
			ann, ok := msg.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("expected msg.Codec for %s to be *messageAnnotations, got %T", msg.ID, msg.Codec)
			}

			if diff := cmp.Diff(test.want, ann.GateExpression()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if !ann.IsGated() {
				t.Error("expected IsGated() to be true")
			}
		})
	}
}

func TestAnnotateMessage_HasConvertedFields(t *testing.T) {
	for _, test := range []struct {
		name   string
		fields []*api.Field
		want   bool
	}{
		{
			name:   "empty_message",
			fields: []*api.Field{},
			want:   false,
		},
		{
			name: "singular_field",
			fields: []*api.Field{
				{Name: "field1", ID: "1", Typez: api.TypezString},
			},
			want: true,
		},
		{
			name: "repeated_field",
			fields: []*api.Field{
				{Name: "field1", ID: "1", Typez: api.TypezString, Repeated: true},
			},
			want: true,
		},
		{
			name: "map_field",
			fields: []*api.Field{
				{Name: "field1", ID: "1", Typez: api.TypezString, Map: true},
			},
			want: true,
		},
		{
			name: "oneof_field",
			fields: []*api.Field{
				{Name: "field1", ID: "1", Typez: api.TypezString, IsOneOf: true},
			},
			want: true,
		},
		{
			name: "map_and_repeated_fields",
			fields: []*api.Field{
				{Name: "labels", ID: "1", Typez: api.TypezString, Map: true},
				{Name: "names", ID: "2", Typez: api.TypezString, Repeated: true},
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := &api.Message{
				Name:    "TestMessage",
				ID:      ".test.TestMessage",
				Package: "test",
				Fields:  test.fields,
			}
			for _, f := range msg.Fields {
				f.Parent = msg
			}
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			model.PackageName = "test"
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			ann, ok := msg.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("expected msg.Codec to be *messageAnnotations, got %T", msg.Codec)
			}
			if ann.HasConvertedFields != test.want {
				t.Errorf("HasConvertedFields = %v, want %v", ann.HasConvertedFields, test.want)
			}
		})
	}
}

func TestAnnotateMessage_ParameterTypeName_Qualification(t *testing.T) {
	deepChild := &api.Message{
		Name:    "DeepChild",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.Parent.Child.DeepChild",
	}
	child := &api.Message{
		Name:     "Child",
		Package:  "google.cloud.secretmanager.v1",
		ID:       ".google.cloud.secretmanager.v1.Parent.Child",
		Messages: []*api.Message{deepChild},
	}
	deepChild.Parent = child
	parent := &api.Message{
		Name:     "Parent",
		Package:  "google.cloud.secretmanager.v1",
		ID:       ".google.cloud.secretmanager.v1.Parent",
		Messages: []*api.Message{child},
	}
	child.Parent = parent

	extDeepChild := &api.Message{
		Name:    "ExtDeepChild",
		Package: "google.type",
		ID:      ".google.type.ExtParent.ExtChild.ExtDeepChild",
	}
	extChild := &api.Message{
		Name:     "ExtChild",
		Package:  "google.type",
		ID:       ".google.type.ExtParent.ExtChild",
		Messages: []*api.Message{extDeepChild},
	}
	extDeepChild.Parent = extChild
	extParent := &api.Message{
		Name:     "ExtParent",
		Package:  "google.type",
		ID:       ".google.type.ExtParent",
		Messages: []*api.Message{extChild},
	}
	extChild.Parent = extParent

	model := api.NewTestAPI([]*api.Message{parent, extParent}, nil, nil)
	model.PackageName = "google.cloud.secretmanager.v1"
	swiftPkg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: "GoogleType", ApiPackage: "google.type"},
			},
		},
	}
	library := &config.Library{
		Name:  "google-cloud-secret-manager",
		Swift: swiftPkg,
	}
	codec := newTestCodec(t, model, library)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		msg     *api.Message
		wantExt string
	}{
		{"local top-level", parent, "Parent"},
		{"local nested", child, "Parent.Child"},
		{"local deep-nested", deepChild, "Parent.Child.DeepChild"},
		{"external top-level", extParent, "GoogleType.ExtParent"},
		{"external nested", extChild, "GoogleType.ExtParent.ExtChild"},
		{"external deep-nested", extDeepChild, "GoogleType.ExtParent.ExtChild.ExtDeepChild"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ann, ok := tc.msg.Codec.(*messageAnnotations)
			if !ok {
				t.Fatalf("expected msg.Codec to be *messageAnnotations, got %T", tc.msg.Codec)
			}
			if ann.ParameterTypeName != tc.wantExt {
				t.Errorf("ParameterTypeName = %q, want %q", ann.ParameterTypeName, tc.wantExt)
			}
		})
	}

	// Also verify module conversion for external packages (e.g. google/type inside GoogleCloudStorage)
	extModuleModel := api.NewTestAPI([]*api.Message{extParent}, nil, nil)
	extModuleModel.PackageName = "google.type"
	extModuleCodec := newTestCodec(t, extModuleModel, &config.Library{
		Name: "google-cloud-storage",
		Swift: &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{Name: "GoogleType", ApiPackage: "google.type"},
				},
			},
			LibraryNameOverride: "GoogleCloudStorage",
		},
	})
	extModuleCodec.Module = true
	extModuleCodec.TargetLibraryName = "GoogleCloudStorage"
	if err := extModuleCodec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	extAnn := extParent.Codec.(*messageAnnotations)
	if extAnn.ParameterTypeName != "GoogleType.ExtParent" {
		t.Errorf("module conversion ParameterTypeName = %q, want %q", extAnn.ParameterTypeName, "GoogleType.ExtParent")
	}
}
