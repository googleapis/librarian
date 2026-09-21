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
			message: func() *api.Message {
				m := api.NewTestMessage("Secret").
					WithFields(
						api.NewTestField("secret_key").WithType(api.TypezString),
					)
				m.Documentation = "A secret message.\nWith two lines."
				return m
			}(),
			want: &messageAnnotations{
				Name:              "Secret",
				DocLines:          []string{"A secret message.", "With two lines."},
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "secretKey",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
		{
			name: "escaped name",
			message: func() *api.Message {
				m := api.NewTestMessage("Protocol")
				m.Documentation = "A message named Protocol."
				return m
			}(),
			want: &messageAnnotations{
				Name:              "Protocol_",
				DocLines:          []string{"A message named Protocol."},
				TypeURL:           "type.googleapis.com/test.Protocol",
				SampleField:       "<placeholder>",
				ParameterTypeName: "Protocol_",
				ProtoTypeName:     "Test_Protocol_",
				ModulePath:        "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
		{
			name: "with oneof",
			message: api.NewTestMessage("WithOneof").
				WithOneOfs(api.NewTestOneOf("choice")),
			want: &messageAnnotations{
				Name:              "WithOneof",
				TypeURL:           "type.googleapis.com/test.WithOneof",
				SampleField:       "<placeholder>",
				ParameterTypeName: "WithOneof",
				ProtoTypeName:     "Test_WithOneof",
				ModulePath:        "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
		{
			name: "with custom json name",
			message: func() *api.Message {
				f := api.NewTestField("secret_key").WithType(api.TypezString)
				f.JSONName = "specialKey"
				return api.NewTestMessage("WithCustomJSON").WithFields(f)
			}(),
			want: &messageAnnotations{
				Name:              "WithCustomJSON",
				TypeURL:           "type.googleapis.com/test.WithCustomJSON",
				SampleField:       "secretKey",
				ParameterTypeName: "WithCustomJSON",
				ProtoTypeName:     "Test_WithCustomJSON",
				ModulePath:        "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
		{
			name: "with pagination",
			message: func() *api.Message {
				pageableItem := api.NewTestField("pageable_item").WithType(api.TypezString).WithRepeated()
				pageableItem.Codec = &fieldAnnotations{Name: "secretKey", BaseFieldType: "SecretKey"}
				m := api.NewTestMessage("WithPagination").
					WithFields(api.NewTestField("secret_key").WithType(api.TypezString))
				m.Pagination = &api.PaginationInfo{
					NextPageToken: api.NewTestField("next_page_token").WithType(api.TypezString),
					PageableItem:  pageableItem,
				}
				return m
			}(),
			want: &messageAnnotations{
				Name:                "WithPagination",
				TypeURL:             "type.googleapis.com/test.WithPagination",
				IsPaginatedResponse: true,
				PageableItemField:   "secretKey",
				PageableItemType:    "SecretKey",
				SampleField:         "secretKey",
				ParameterTypeName:   "WithPagination",
				ProtoTypeName:       "Test_WithPagination",
				ModulePath:          "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleGax"}, {Module: "GoogleWKT"}},
		},
		{
			name: "service placeholder",
			message: func() *api.Message {
				m := api.NewTestMessage("Service")
				m.ServicePlaceholder = true
				return m
			}(),
			want: &messageAnnotations{
				Name:              "Service",
				TypeURL:           "type.googleapis.com/test.Service",
				SampleField:       "<placeholder>",
				ParameterTypeName: "ServiceClient",
				PlaceholderName:   "ServiceClient",
				ProtoTypeName:     "Test_Service",
				ModulePath:        "",
			},
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.message}, nil, nil)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
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
			wantImports: []*dependencyImport{{Module: "GoogleWKT"}},
		},
		{
			name:        "with spi attribute",
			message:     api.NewTestMessage("Secret"),
			config:      withSpiConfig,
			wantImports: []*dependencyImport{{Module: "GoogleWKT", Attributes: []string{"@_spi(Test001)"}}},
		},
		{
			name: "with pagination",
			message: api.NewTestMessage("WithPagination").WithPagination(
				api.NewTestField("next_page_token").WithType(api.TypezString),
				api.NewTestField("pageable_item").WithType(api.TypezString).WithRepeated(),
			),
			config:      noSpiConfig,
			wantImports: []*dependencyImport{{Module: "GoogleGax"}, {Module: "GoogleWKT"}},
		},
		{
			name: "with pagination and attributes",
			message: api.NewTestMessage("WithPagination").WithPagination(
				api.NewTestField("next_page_token").WithType(api.TypezString),
				api.NewTestField("pageable_item").WithType(api.TypezString).WithRepeated(),
			),
			config: withSpiConfig,
			wantImports: []*dependencyImport{
				{Module: "GoogleGax", Attributes: []string{"@_spi(Test002)"}},
				{Module: "GoogleWKT", Attributes: []string{"@_spi(Test001)"}},
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
	mapMessage := api.NewTestMessage("map<string, bytes>").
		WithID("$map<string, bytes>").
		WithFields(
			api.NewTestField("key").WithType(api.TypezString),
			api.NewTestField("value").WithType(api.TypezBytes),
		)
	mapMessage.IsMap = true

	for _, test := range []struct {
		name    string
		message *api.Message
		want    *messageAnnotations
	}{
		{
			name: "simple",
			message: api.NewTestMessage("Secret").
				WithFields(api.NewTestField("field").WithType(api.TypezString)),
			want: &messageAnnotations{
				Name:              "Secret",
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "field",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
		},
		{
			name: "required",
			message: api.NewTestMessage("Secret").
				WithFields(api.NewTestField("field").WithType(api.TypezBytes)),
			want: &messageAnnotations{
				Name:              "Secret",
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "field",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
		},
		{
			name: "optional",
			message: api.NewTestMessage("Secret").
				WithFields(api.NewTestField("field").WithType(api.TypezBytes).WithOptional()),
			want: &messageAnnotations{
				Name:              "Secret",
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "field",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
		},
		{
			name: "repeated",
			message: api.NewTestMessage("Secret").
				WithFields(api.NewTestField("field").WithType(api.TypezBytes).WithRepeated()),
			want: &messageAnnotations{
				Name:              "Secret",
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "field",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
		},
		{
			name: "map",
			message: api.NewTestMessage("Secret").
				WithFields(api.NewTestField("field").WithMessageType(mapMessage).WithMap()),
			want: &messageAnnotations{
				Name:              "Secret",
				TypeURL:           "type.googleapis.com/test.Secret",
				SampleField:       "field",
				ParameterTypeName: "Secret",
				ProtoTypeName:     "Test_Secret",
				ModulePath:        "",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{test.message}, nil, nil)
			model.AddMessage(mapMessage)
			codec := newTestCodec(t, model, nil)
			codec.UrlSafeForBytes = true
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
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
			service: api.NewTestService("Service"),
			request: func() *api.Message {
				m := api.NewTestMessage("getRequest").WithID(".test.Service.getRequest")
				m.SyntheticRequest = true
				return m
			}(),
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
			service: api.NewTestService("Protocol"),
			request: func() *api.Message {
				m := api.NewTestMessage("listRequest").WithID(".test.Protocol.listRequest")
				m.SyntheticRequest = true
				return m
			}(),
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
			servicePlaceholder := api.NewTestMessage(test.service.Name).
				WithPackage(test.service.Package).
				WithID(test.service.ID)
			servicePlaceholder.ServicePlaceholder = true
			test.request.Parent = servicePlaceholder
			servicePlaceholder.Messages = append(servicePlaceholder.Messages, test.request)
			model := api.NewTestAPI([]*api.Message{servicePlaceholder}, nil, []*api.Service{test.service})
			model.AddMessage(test.request)
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.request.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMessage_Pagination(t *testing.T) {
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
	inputType := api.NewTestMessage("ListSecretsRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(pageSizeField, pageTokenField)

	secretType := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
	itemField := api.NewTestField("secrets").
		WithMessageType(secretType).
		WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	outputType := api.NewTestMessage("ListSecretsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithFields(itemField, nextPageTokenField).
		WithPagination(nextPageTokenField, itemField)

	method := api.NewTestMethod("ListSecrets").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets")).
		WithPagination(pageTokenField)

	iam := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithMethods(method)

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
	if diff := cmp.Diff(wantRequest, gotRequest, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantRequestImports := []*dependencyImport{{Module: "GoogleWKT"}}
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
	if diff := cmp.Diff(wantResponse, gotResponse, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wantResponseImports := []*dependencyImport{{Module: "GoogleGax"}, {Module: "GoogleWKT"}}
	if diff := cmp.Diff(wantResponseImports, gotResponse.MessageImports()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMessage_RecursiveNested(t *testing.T) {
	secretType := api.NewTestMessage("Secret").WithPackage("google.cloud.secretmanager.v1")
	itemField := api.NewTestField("secrets").
		WithMessageType(secretType).
		WithRepeated()
	nextPageTokenField := api.NewTestField("next_page_token").WithType(api.TypezString)
	nestedOutputType := api.NewTestMessage("ListSecretsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithID(".google.cloud.secretmanager.v1.OuterMessage.ListSecretsResponse").
		WithFields(itemField, nextPageTokenField).
		WithPagination(nextPageTokenField, itemField)

	outerMessage := api.NewTestMessage("OuterMessage").
		WithPackage("google.cloud.secretmanager.v1")
	outerMessage.Messages = []*api.Message{nestedOutputType}
	nestedOutputType.Parent = outerMessage

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
	if diff := cmp.Diff(wantOuter, gotOuter, cmpopts.IgnoreFields(messageAnnotations{}, "Model", "DependsOn")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantImports := []*dependencyImport{{Module: "GoogleGax"}, {Module: "GoogleWKT"}}
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

func TestAnnotateMessage_ParameterTypeName_Qualification(t *testing.T) {
	deepChild := api.NewTestMessage("DeepChild").
		WithPackage("google.cloud.secretmanager.v1").
		WithID(".google.cloud.secretmanager.v1.Parent.Child.DeepChild")
	child := api.NewTestMessage("Child").
		WithPackage("google.cloud.secretmanager.v1").
		WithID(".google.cloud.secretmanager.v1.Parent.Child")
	child.Messages = []*api.Message{deepChild}
	deepChild.Parent = child
	parent := api.NewTestMessage("Parent").
		WithPackage("google.cloud.secretmanager.v1")
	parent.Messages = []*api.Message{child}
	child.Parent = parent

	extDeepChild := api.NewTestMessage("ExtDeepChild").
		WithPackage("google.type").
		WithID(".google.type.ExtParent.ExtChild.ExtDeepChild")
	extChild := api.NewTestMessage("ExtChild").
		WithPackage("google.type").
		WithID(".google.type.ExtParent.ExtChild")
	extChild.Messages = []*api.Message{extDeepChild}
	extDeepChild.Parent = extChild
	extParent := api.NewTestMessage("ExtParent").
		WithPackage("google.type")
	extParent.Messages = []*api.Message{extChild}
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
