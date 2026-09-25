// Copyright 2024 Google LLC
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

package parser

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/types/known/apipb"
)

func TestOpenAPI_AllOf(t *testing.T) {
	// A message with AllOf and its transitive closure of dependent messages.
	const messageWithAllOf = `
      "Automatic": {
        "description": "A replication policy that replicates the Secret payload without any restrictions.",
        "type": "object",
        "properties": {
          "customerManagedEncryption": {
            "description": "Optional. The customer-managed encryption configuration of the Secret.",
            "allOf": [{
              "$ref": "#/components/schemas/CustomerManagedEncryption"
            }]
          }
        }
      },
      "CustomerManagedEncryption": {
        "description": "Configuration for encrypting secret payloads using customer-managed\nencryption keys (CMEK).",
        "type": "object",
        "properties": {
          "kmsKeyName": {
            "description": "Required. The resource name of the Cloud KMS CryptoKey used to encrypt secret payloads.",
            "type": "string"
          }
        },
        "required": [
          "kmsKeyName"
        ]
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithAllOf + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	want := sample.Automatic()
	want.Package = ""
	message := test.Message(want.ID)
	if message == nil {
		t.Errorf("missing message in MessageByID index")
		return
	}
	apitest.CheckMessage(t, message, want)
}

func newTestOpenAPIFakeMessage(fields ...*api.Field) *api.Message {
	return api.NewTestMessage("Fake").
		WithPackage("").
		WithID("..Fake").
		WithDocumentation("A test message.").
		WithFields(fields...)
}

func TestOpenAPI_BasicTypes(t *testing.T) {
	// A message with basic types.
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fBool":      { "type": "boolean" },
          "fInt64":     { "type": "integer", "format": "int64" },
          "fInt32":     { "type": "integer", "format": "int32" },
          "fUInt32":    { "type": "integer", "format": "int32", "minimum": 0 },
          "fFloat":     { "type": "number", "format": "float" },
          "fDouble":    { "type": "number", "format": "double" },
          "fString":    { "type": "string" },
          "fOptional":  { "type": "string" },
          "fSInt64":    { "type": "string", "format": "int64" },
          "fSUInt64":   { "type": "string", "format": "int64", "minimum": 0 },
          "fDuration":  { "type": "string", "format": "google-duration" },
          "fTimestamp": { "type": "string", "format": "date-time" },
          "fFieldMask": { "type": "string", "format": "google-fieldmask" },
          "fBytes":     { "type": "string", "format": "byte" }
        },
        "required": [
            "fBool", "fInt64", "fInt32", "fUInt32",
            "fFloat", "fDouble",
            "fString",
            "fSInt64", "fSUInt64",
            "fDuration", "fTimestamp", "fFieldMask", "fBytes"
        ]
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}

	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	message := test.Message("..Fake")
	if message == nil {
		t.Errorf("missing message in MessageByID index")
		return
	}
	apitest.CheckMessage(t, message, newTestOpenAPIFakeMessage(
		api.NewTestField("fBool").WithType(api.TypezBool).WithTypezID("bool"),
		api.NewTestField("fInt64").WithType(api.TypezInt64).WithTypezID("int64"),
		api.NewTestField("fInt32").WithType(api.TypezInt32).WithTypezID("int32"),
		api.NewTestField("fUInt32").WithType(api.TypezUint32).WithTypezID("uint32").WithJSONName("fUInt32"),
		api.NewTestField("fFloat").WithType(api.TypezFloat).WithTypezID("float"),
		api.NewTestField("fDouble").WithType(api.TypezDouble).WithTypezID("double"),
		api.NewTestField("fString").WithType(api.TypezString).WithTypezID("string"),
		api.NewTestField("fOptional").WithType(api.TypezString).WithTypezID("string").WithOptional(),
		api.NewTestField("fSInt64").WithType(api.TypezInt64).WithTypezID("int64").WithJSONName("fSInt64"),
		api.NewTestField("fSUInt64").WithType(api.TypezUint64).WithTypezID("uint64").WithJSONName("fSUInt64"),
		api.NewTestField("fDuration").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Duration").WithOptional(),
		api.NewTestField("fTimestamp").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Timestamp").WithOptional(),
		api.NewTestField("fFieldMask").WithType(api.TypezMessage).WithTypezID(".google.protobuf.FieldMask").WithOptional(),
		api.NewTestField("fBytes").WithType(api.TypezBytes).WithTypezID("bytes"),
	))
}

func TestOpenAPI_ArrayTypes(t *testing.T) {
	// A message with basic types.
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fBool":      { "type": "array", "items": { "type": "boolean" }},
          "fInt64":     { "type": "array", "items": { "type": "integer", "format": "int64" }},
          "fInt32":     { "type": "array", "items": { "type": "integer", "format": "int32" }},
          "fUInt32":    { "type": "array", "items": { "type": "integer", "format": "int32", "minimum": 0 }},
          "fString":    { "type": "array", "items": { "type": "string" }},
          "fSInt64":    { "type": "array", "items": { "type": "string", "format": "int64" }},
          "fSUInt64":   { "type": "array", "items": { "type": "string", "format": "int64", "minimum": 0 }},
          "fDuration":  { "type": "array", "items": { "type": "string", "format": "google-duration" }},
          "fTimestamp": { "type": "array", "items": { "type": "string", "format": "date-time" }},
          "fFieldMask": { "type": "array", "items": { "type": "string", "format": "google-fieldmask" }},
          "fBytes":     { "type": "array", "items": { "type": "string", "format": "byte" }},
        }
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	message := test.Message("..Fake")
	if message == nil {
		t.Errorf("missing message in MessageByID index")
		return
	}
	apitest.CheckMessage(t, message, newTestOpenAPIFakeMessage(
		api.NewTestField("fBool").WithType(api.TypezBool).WithTypezID("bool").WithRepeated(),
		api.NewTestField("fInt64").WithType(api.TypezInt64).WithTypezID("int64").WithRepeated(),
		api.NewTestField("fInt32").WithType(api.TypezInt32).WithTypezID("int32").WithRepeated(),
		api.NewTestField("fUInt32").WithType(api.TypezUint32).WithTypezID("uint32").WithJSONName("fUInt32").WithRepeated(),
		api.NewTestField("fString").WithType(api.TypezString).WithTypezID("string").WithRepeated(),
		api.NewTestField("fSInt64").WithType(api.TypezInt64).WithTypezID("int64").WithJSONName("fSInt64").WithRepeated(),
		api.NewTestField("fSUInt64").WithType(api.TypezUint64).WithTypezID("uint64").WithJSONName("fSUInt64").WithRepeated(),
		api.NewTestField("fDuration").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Duration").WithRepeated(),
		api.NewTestField("fTimestamp").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Timestamp").WithRepeated(),
		api.NewTestField("fFieldMask").WithType(api.TypezMessage).WithTypezID(".google.protobuf.FieldMask").WithRepeated(),
		api.NewTestField("fBytes").WithType(api.TypezBytes).WithTypezID("bytes").WithRepeated(),
	))
}

func TestOpenAPI_SimpleObject(t *testing.T) {
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fObject"     : { "type": "object", "description": "An object field.", "allOf": [{ "$ref": "#/components/schemas/Foo" }] },
          "fObjectArray": { "type": "array",  "description": "An object array field.", "items": [{ "$ref": "#/components/schemas/Bar" }] }
        }
      },
      "Foo": {
        "description": "Must have a Foo.",
        "type": "object",
        "properties": {}
      },
      "Bar": {
        "description": "Must have a Bar.",
        "type": "object",
        "properties": {}
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	apitest.CheckMessage(t, test.Messages[0], newTestOpenAPIFakeMessage(
		api.NewTestField("fObject").
			WithDocumentation("An object field.").
			WithType(api.TypezMessage).
			WithTypezID("..Foo").
			WithOptional(),
		api.NewTestField("fObjectArray").
			WithDocumentation("An object array field.").
			WithType(api.TypezMessage).
			WithTypezID("..Bar").
			WithRepeated(),
	))
}

func TestOpenAPI_Any(t *testing.T) {
	// A message with basic types.
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fMap":       { "type": "object", "additionalProperties": { "description": "Test Only." }}
        }
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Errorf("Error in makeAPI() %q", err)
	}

	apitest.CheckMessage(t, test.Messages[0], newTestOpenAPIFakeMessage(
		api.NewTestField("fMap").WithType(api.TypezMessage).WithTypezID(".google.protobuf.Any").WithOptional(),
	))
}

func TestOpenAPI_MapString(t *testing.T) {
	// A message with basic types.
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fMap":     { "type": "object", "additionalProperties": { "type": "string" }},
          "fMapS32":  { "type": "object", "additionalProperties": { "type": "string", "format": "int32" }},
          "fMapS64":  { "type": "object", "additionalProperties": { "type": "string", "format": "int64" }}
        }
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatal(err)
	}

	wantMap := api.NewTestMessage("Fake").
		WithPackage("").
		WithID("..Fake").
		WithDocumentation("A test message.").
		WithFields(
			api.NewTestField("fMap").WithType(api.TypezMessage).WithTypezID("$map<string, string>").WithMap(),
			api.NewTestField("fMapS32").WithType(api.TypezMessage).WithTypezID("$map<string, int32>").WithMap(),
			api.NewTestField("fMapS64").WithType(api.TypezMessage).WithTypezID("$map<string, int64>").WithMap(),
		)
	for _, f := range wantMap.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, test.Messages[0], wantMap)
}

func TestOpenAPI_MapInteger(t *testing.T) {
	// A message with basic types.
	const messageWithBasicTypes = `
      "Fake": {
        "description": "A test message.",
        "type": "object",
        "properties": {
          "fMapI32": { "type": "object", "additionalProperties": { "type": "integer", "format": "int32" }},
          "fMapI64": { "type": "object", "additionalProperties": { "type": "integer", "format": "int64" }}
        }
      },
`
	contents := []byte(openAPISingleMessagePreamble + messageWithBasicTypes + openAPISingleMessageTrailer)
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Errorf("Error in makeAPI() %q", err)
	}

	wantMapInteger := api.NewTestMessage("Fake").
		WithPackage("").
		WithID("..Fake").
		WithDocumentation("A test message.").
		WithFields(
			api.NewTestField("fMapI32").WithType(api.TypezMessage).WithTypezID("$map<string, int32>").WithMap(),
			api.NewTestField("fMapI64").WithType(api.TypezMessage).WithTypezID("$map<string, int64>").WithMap(),
		)
	for _, f := range wantMapInteger.Fields {
		f.Parent = nil
	}
	apitest.CheckMessage(t, test.Messages[0], wantMapInteger)
}

func openapiSecretManagerAPI(t *testing.T) *api.API {
	t.Helper()
	contents, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}
	api.UpdateMethodPagination(nil, test)
	return test
}

func TestOpenAPI_MakeAPI(t *testing.T) {
	test := openapiSecretManagerAPI(t)

	location := test.Message("..Location")
	if location == nil {
		t.Errorf("missing message (Location) in MessageByID index")
		return
	}
	apitest.CheckMessage(t, location, &api.Message{
		Documentation: "A resource that represents a Google Cloud location.",
		Name:          "Location",
		ID:            "..Location",
		Fields: []*api.Field{
			{
				Name:          "name",
				ID:            "..Location.name",
				JSONName:      "name",
				Documentation: "Resource name for the location, which may vary between implementations." + "\nFor example: `\"projects/example-project/locations/us-east1\"`",
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "locationId",
				ID:            "..Location.locationId",
				JSONName:      "locationId",
				Documentation: "The canonical id for this location. For example: `\"us-east1\"`.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "displayName",
				ID:            "..Location.displayName",
				JSONName:      "displayName",
				Documentation: `The friendly name for this location, typically a nearby city name.` + "\n" + `For example, "Tokyo".`,
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "labels",
				ID:            "..Location.labels",
				JSONName:      "labels",
				Documentation: "Cross-service attributes for the location. For example\n\n    {\"cloud.googleapis.com/region\": \"us-east1\"}",
				Typez:         api.TypezMessage,
				TypezID:       "$map<string, string>",
				Optional:      false,
				Map:           true,
			},
			{
				Name:          "metadata",
				ID:            "..Location.metadata",
				JSONName:      "metadata",
				Documentation: `Service-specific metadata. For example the available capacity at the given` + "\n" + `location.`,
				Typez:         api.TypezMessage,
				TypezID:       ".google.protobuf.Any",
				Optional:      true,
			},
		},
	})

	listLocationsResponse := test.Message("..ListLocationsResponse")
	if listLocationsResponse == nil {
		t.Errorf("missing message (ListLocationsResponse) in MessageByID index")
		return
	}
	apitest.CheckMessage(t, listLocationsResponse, &api.Message{
		Documentation: "The response message for Locations.ListLocations.",
		Name:          "ListLocationsResponse",
		ID:            "..ListLocationsResponse",
		Fields: []*api.Field{
			{
				Name:          "locations",
				ID:            "..ListLocationsResponse.locations",
				JSONName:      "locations",
				Documentation: "A list of locations that matches the specified filter in the request.",
				Typez:         api.TypezMessage,
				TypezID:       "..Location",
				Repeated:      true,
			},
			{
				Name:          "nextPageToken",
				ID:            "..ListLocationsResponse.nextPageToken",
				JSONName:      "nextPageToken",
				Documentation: "The standard List next-page token.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
		},
		Pagination: &api.PaginationInfo{
			NextPageToken: &api.Field{
				Name:          "nextPageToken",
				ID:            "..ListLocationsResponse.nextPageToken",
				JSONName:      "nextPageToken",
				Documentation: "The standard List next-page token.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
			PageableItem: &api.Field{
				Name:          "locations",
				ID:            "..ListLocationsResponse.locations",
				JSONName:      "locations",
				Documentation: "A list of locations that matches the specified filter in the request.",
				Typez:         api.TypezMessage,
				TypezID:       "..Location",
				Repeated:      true,
			},
		},
	})

	// This is a synthetic message, the OpenAPI spec does not contain requests
	// messages for messages without a body.
	want := &api.Message{
		Name:             "ListLocationsRequest",
		ID:               "..Service.ListLocationsRequest",
		Documentation:    "Synthetic request message for the [ListLocations()][.Service.ListLocations] method.",
		SyntheticRequest: true,
		Fields: []*api.Field{
			{
				Name:          "project",
				ID:            "..Service.ListLocationsRequest.project",
				JSONName:      "project",
				Documentation: "The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/locations`.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Behavior:      []api.FieldBehavior{api.FieldBehaviorRequired},
			},
			{
				Name:     "filter",
				ID:       "..Service.ListLocationsRequest.filter",
				JSONName: "filter",
				Documentation: "A filter to narrow down results to a preferred subset." +
					"\nThe filtering language accepts strings like `\"displayName=tokyo" +
					"\"`, and\nis documented in more detail in [AIP-160](https://google" +
					".aip.dev/160).",
				Typez:    api.TypezString,
				TypezID:  "string",
				Optional: true,
			},
			{
				Name:          "pageSize",
				ID:            "..Service.ListLocationsRequest.pageSize",
				JSONName:      "pageSize",
				Documentation: "The maximum number of results to return.\nIf not set, the service selects a default.",
				Typez:         api.TypezInt32,
				TypezID:       "int32",
				Optional:      true,
			},
			{
				Name:          "pageToken",
				ID:            "..Service.ListLocationsRequest.pageToken",
				JSONName:      "pageToken",
				Documentation: "A page token received from the `next_page_token` field in the response.\nSend that page token to receive the subsequent page.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Optional:      true,
			},
		},
	}
	listLocationsRequest := test.Message(want.ID)
	if listLocationsRequest == nil {
		t.Errorf("missing message (%s) in MessageByID index", want.ID)
		return
	}
	apitest.CheckMessage(t, listLocationsRequest, want)

	// This message has a weirdly named field that gets tricky to serialize.
	sp := sample.SecretPayload()
	got := test.Message(sp.ID)
	if got == nil {
		t.Errorf("missing message (SecretPayload) in MessageByID index")
		return
	}
	apitest.CheckMessage(t, got, sp)

	service := test.Service("..Service")
	if service == nil {
		t.Errorf("missing service (Service) in ServiceByID index")
		return
	}

	wantService := sample.Service()
	wantService.Package = ""
	wantService.Name = "Service"
	wantService.ID = "..Service"
	if diff := cmp.Diff(wantService, service, cmpopts.IgnoreFields(api.Service{}, "Methods")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	apitest.CheckMethod(t, service, "ListLocations", &api.Method{
		Name:          "ListLocations",
		ID:            "..Service.ListLocations",
		Documentation: "Lists information about the supported locations for this service.",
		InputTypeID:   "..Service.ListLocationsRequest",
		OutputTypeID:  "..ListLocationsResponse",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("locations"),
					QueryParameters: map[string]bool{
						"filter":    true,
						"pageSize":  true,
						"pageToken": true,
					},
				},
			},
		},
		Pagination: &api.Field{
			Name:          "pageToken",
			ID:            "..Service.ListLocationsRequest.pageToken",
			JSONName:      "pageToken",
			Documentation: "A page token received from the `next_page_token` field in the response.\nSend that page token to receive the subsequent page.",
			Typez:         api.TypezString,
			TypezID:       "string",
			Optional:      true,
		},
	})

	cs := sample.MethodCreate()
	apitest.CheckMethod(t, service, cs.Name, cs)

	asv := sample.MethodAddSecretVersion()
	apitest.CheckMethod(t, service, asv.Name, asv)
}

func TestOpenAPI_ServicePlaceholder(t *testing.T) {
	test := openapiSecretManagerAPI(t)
	want := &api.Message{
		Name:               "Service",
		ID:                 "..Service",
		Package:            "",
		Documentation:      "Synthetic messages for the [Service][.Service] service.",
		ServicePlaceholder: true,
	}
	got := test.Message("..Service")
	if got == nil {
		t.Errorf("missing service placeholder message in MessageById index")
		return
	}
	apitest.CheckMessage(t, got, want)
}

func TestOpenAPI_MakeApiWithServiceConfig(t *testing.T) {
	contents, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	got, err := makeAPIForOpenAPI(sample.ServiceConfig(), model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}
	want := sample.API()
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.API{}, "Services", "Messages", "Enums"), cmpopts.IgnoreUnexported(api.API{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

}

func TestOpenAPI_MakeApiServiceConfigOverridesDescription(t *testing.T) {
	contents, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	serviceConfig := sample.ServiceConfig()
	serviceConfig.Documentation.Summary = "Test Only - Override Description."
	want := sample.API()
	want.Description = serviceConfig.Documentation.Summary
	got, err := makeAPIForOpenAPI(serviceConfig, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.API{}, "Services", "Messages", "Enums"), cmpopts.IgnoreUnexported(api.API{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

}

func TestOpenAPI_SyntheticMessageWithExistingBody(t *testing.T) {
	contents, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	want := api.NewTestMessage("Service").
		WithPackage("").
		WithID("..Service").
		WithDocumentation("Synthetic messages for the [Service][.Service] service.")
	want.ServicePlaceholder = true
	got := test.Message(want.ID)
	if got == nil {
		t.Errorf("missing message (%s) in MessageByID index", want.ID)
		return
	}
	apitest.CheckMessage(t, got, want)

	// Methods that share a body should create separate requests.
	want = api.NewTestMessage("SetIamPolicyByProjectAndLocationAndSecretRequest").
		WithPackage("").
		WithID("..Service.SetIamPolicyByProjectAndLocationAndSecretRequest").
		WithDocumentation("Synthetic request message for the [SetIamPolicyByProjectAndLocationAndSecret()][.Service.SetIamPolicyByProjectAndLocationAndSecret] method.").
		WithFields(
			api.NewTestField("project").
				WithDocumentation("The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/locations/{location}/secrets/{secret}:setIamPolicy`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("location").
				WithDocumentation("The `{location}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/locations/{location}/secrets/{secret}:setIamPolicy`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("secret").
				WithDocumentation("The `{secret}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/locations/{location}/secrets/{secret}:setIamPolicy`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("body").
				WithDocumentation("The request body.").
				WithType(api.TypezMessage).
				WithTypezID("..SetIamPolicyRequest").
				WithOptional(),
		)
	for _, f := range want.Fields {
		f.Parent = nil
	}
	want.SyntheticRequest = true
	got = test.Message(want.ID)
	if got == nil {
		t.Errorf("missing message (%s) in MessageByID index", want.ID)
		return
	}
	apitest.CheckMessage(t, got, want)

	want = api.NewTestMessage("SetIamPolicyRequest").
		WithPackage("").
		WithID("..Service.SetIamPolicyRequest").
		WithDocumentation("Synthetic request message for the [SetIamPolicy()][.Service.SetIamPolicy] method.").
		WithFields(
			api.NewTestField("project").
				WithDocumentation("The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/secrets/{secret}:setIamPolicy`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("secret").
				WithDocumentation("The `{secret}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/secrets/{secret}:setIamPolicy`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("body").
				WithDocumentation("The request body.").
				WithType(api.TypezMessage).
				WithTypezID("..SetIamPolicyRequest").
				WithOptional(),
		)
	for _, f := range want.Fields {
		f.Parent = nil
	}
	want.SyntheticRequest = true
	got = test.Message(want.ID)
	if got == nil {
		t.Errorf("missing message (%s) in MessageByID index", want.ID)
		return
	}
	apitest.CheckMessage(t, got, want)
}

func TestOpenAPI_Pagination(t *testing.T) {
	contents, err := os.ReadFile("testdata/pagination_openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}
	api.UpdateMethodPagination(nil, test)

	service := test.Service("..Service")
	if service == nil {
		t.Errorf("missing service (Service) in ServiceByID index")
		return
	}
	apitest.CheckService(t, service, &api.Service{
		Name: "Service",
		ID:   "..Service",
		Methods: []*api.Method{
			{
				Name:         "ListFoos",
				ID:           "..Service.ListFoos",
				InputTypeID:  "..Service.ListFoosRequest",
				OutputTypeID: "..ListFoosResponse",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: (&api.PathTemplate{}).
								WithLiteral("v1").
								WithLiteral("projects").
								WithVariableNamed("project").
								WithLiteral("foos"),
							QueryParameters: map[string]bool{"pageSize": true, "pageToken": true},
						},
					},
				},
				Pagination: &api.Field{
					Name:          "pageToken",
					ID:            "..Service.ListFoosRequest.pageToken",
					JSONName:      "pageToken",
					Documentation: "The `{pageToken}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/foos`.",
					Typez:         api.TypezString,
					TypezID:       "string",
					Optional:      true,
				},
			},
		},
	})
	resp := test.Message("..ListFoosResponse")
	if resp == nil {
		t.Errorf("missing message (ListFoosResponse) in MessageByID index")
		return
	}
	apitest.CheckMessage(t, resp, &api.Message{
		Name: "ListFoosResponse",
		ID:   "..ListFoosResponse",
		Fields: []*api.Field{
			{
				Name:     "nextPageToken",
				ID:       "..ListFoosResponse.nextPageToken",
				Typez:    9,
				TypezID:  "string",
				JSONName: "nextPageToken",
				Optional: true,
			},
			{
				Name:     "secrets",
				ID:       "..ListFoosResponse.secrets",
				Typez:    api.TypezMessage,
				TypezID:  "..Foo",
				JSONName: "secrets",
				Repeated: true,
			},
		},
		Pagination: &api.PaginationInfo{
			NextPageToken: &api.Field{
				Name:     "nextPageToken",
				ID:       "..ListFoosResponse.nextPageToken",
				Typez:    9,
				TypezID:  "string",
				JSONName: "nextPageToken",
				Optional: true,
			},
			PageableItem: &api.Field{
				Name:     "secrets",
				ID:       "..ListFoosResponse.secrets",
				Typez:    api.TypezMessage,
				TypezID:  "..Foo",
				JSONName: "secrets",
				Repeated: true,
			},
		},
	})
}

func TestOpenAPI_AutoPopulated(t *testing.T) {
	serviceConfig := &serviceconfig.Service{
		Name:  "test",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
		},
		Apis: []*apipb.Api{
			{
				Name: "test.TestService",
			},
		},
		Publishing: &annotations.Publishing{
			MethodSettings: []*annotations.MethodSettings{
				{
					Selector: "test.TestService.CreateFoo",
					AutoPopulatedFields: []string{
						"requestId",
						"requestIdExplicitlyNotRequired",
						"notRequestIdRequired",
						"notRequestIdMissingFormat",
					},
				},
			},
		},
	}

	contents, err := os.ReadFile("testdata/auto_populated_openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(serviceConfig, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	request_id := &api.Field{
		Name:          "requestId",
		ID:            ".test.TestService.CreateFooRequest.requestId",
		JSONName:      "requestId",
		Documentation: "Test-only Description",
		Typez:         api.TypezString,
		TypezID:       "string",
		Optional:      true,
		AutoPopulated: true,
	}
	request_id_explicit := &api.Field{
		Name:          "requestIdExplicitlyNotRequired",
		ID:            ".test.TestService.CreateFooRequest.requestIdExplicitlyNotRequired",
		JSONName:      "requestIdExplicitlyNotRequired",
		Documentation: "Test-only Description",
		Typez:         api.TypezString,
		TypezID:       "string",
		Optional:      true,
		AutoPopulated: true,
	}
	wantMessage := &api.Message{
		Name:             "CreateFooRequest",
		ID:               ".test.TestService.CreateFooRequest",
		Package:          "test",
		Documentation:    "Synthetic request message for the [CreateFoo()][test.TestService.CreateFoo] method.",
		SyntheticRequest: true,
		Fields: []*api.Field{
			{
				Name:          "project",
				ID:            ".test.TestService.CreateFooRequest.project",
				JSONName:      "project",
				Documentation: "The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/foos`.",
				Typez:         api.TypezString,
				TypezID:       "string",
				Behavior:      []api.FieldBehavior{api.FieldBehaviorRequired},
			},
			{
				Name:          "fooId",
				ID:            ".test.TestService.CreateFooRequest.fooId",
				JSONName:      "fooId",
				Documentation: "Test-only Description",
				Typez:         api.TypezString,
				TypezID:       "string",
				Behavior:      []api.FieldBehavior{api.FieldBehaviorRequired},
			},
			request_id,
			request_id_explicit,
			{
				Name:          "notRequestIdRequired",
				ID:            ".test.TestService.CreateFooRequest.notRequestIdRequired",
				Documentation: "Test-only Description",
				Typez:         api.TypezString,
				TypezID:       "string",
				JSONName:      "notRequestIdRequired",
				Behavior:      []api.FieldBehavior{api.FieldBehaviorRequired},
			},
			{
				Name:          "notRequestIdMissingFormat",
				ID:            ".test.TestService.CreateFooRequest.notRequestIdMissingFormat",
				Documentation: "Test-only Description",
				Typez:         api.TypezString,
				TypezID:       "string",
				JSONName:      "notRequestIdMissingFormat",
				Optional:      true,
			},
			{
				Name:          "notRequestIdMissingServiceConfig",
				ID:            ".test.TestService.CreateFooRequest.notRequestIdMissingServiceConfig",
				Documentation: "Test-only Description",
				Typez:         api.TypezString,
				TypezID:       "string",
				JSONName:      "notRequestIdMissingServiceConfig",
				Optional:      true,
				// This just denotes that the field is eligible
				// to be auto-populated
				AutoPopulated: true,
			},
		},
	}
	message := test.Message(wantMessage.ID)
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", wantMessage.ID)
	}
	apitest.CheckMessage(t, message, wantMessage)

	method := test.Method(".test.TestService.CreateFoo")
	if method == nil {
		t.Fatalf("Cannot find method %s in API State", ".test.TestService.CreateFoo")
	}
	wantField := []*api.Field{request_id, request_id_explicit}
	if diff := cmp.Diff(wantField, method.AutoPopulated); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestOpenAPI_Deprecated(t *testing.T) {
	contents, err := os.ReadFile("testdata/deprecated_openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	model, err := createDocModel(contents)
	if err != nil {
		t.Fatal(err)
	}
	test, err := makeAPIForOpenAPI(nil, model)
	if err != nil {
		t.Fatalf("Error in makeAPI() %q", err)
	}

	service := test.Service("..Service")
	if service == nil {
		t.Errorf("cannot find service %s in model", "..Service.ListFoos")
		return
	}
	apitest.CheckMethod(t, service, "RpcA", &api.Method{
		Name:         "RpcA",
		ID:           "..Service.RpcA",
		InputTypeID:  "..Service.RpcARequest",
		OutputTypeID: "..Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("rpc").
						WithLiteral("a"),
					QueryParameters: map[string]bool{"filter": true},
				},
			},
		},
	})

	apitest.CheckMethod(t, service, "RpcB", &api.Method{
		Name:         "RpcB",
		ID:           "..Service.RpcB",
		Deprecated:   true,
		InputTypeID:  "..Service.RpcBRequest",
		OutputTypeID: "..Response",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: (&api.PathTemplate{}).
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("rpc").
						WithLiteral("b"),
					QueryParameters: map[string]bool{},
				},
			},
		},
	})

	response := test.Message("..Response")
	if response == nil {
		t.Errorf("cannot find message %s", "..Response")
		return
	}
	apitest.CheckMessage(t, response, &api.Message{
		Name: "Response",
		ID:   "..Response",
		Fields: []*api.Field{
			{
				Name:     "name",
				ID:       "..Response.name",
				Typez:    api.TypezString,
				TypezID:  "string",
				JSONName: "name",
				Optional: true,
			},
			{
				Name:       "other",
				ID:         "..Response.other",
				Typez:      api.TypezString,
				TypezID:    "string",
				JSONName:   "other",
				Deprecated: true,
				Optional:   true,
			},
		},
	})

	deprecatedMessage := test.Message("..DeprecatedMessage")
	if deprecatedMessage == nil {
		t.Errorf("cannot find message %s", "..DeprecatedMessage")
		return
	}
	apitest.CheckMessage(t, deprecatedMessage, &api.Message{
		Name:       "DeprecatedMessage",
		ID:         "..DeprecatedMessage",
		Deprecated: true,
		Fields: []*api.Field{
			{
				Name:     "name",
				ID:       "..DeprecatedMessage.name",
				Typez:    api.TypezString,
				TypezID:  "string",
				JSONName: "name",
				Optional: true,
			},
		},
	})
}

func TestOpenAPI_ParseBadFiles(t *testing.T) {
	for _, cfg := range []*ModelConfig{
		{SpecificationSource: "-invalid-file-name-", ServiceConfig: secretManagerYamlFullPath},
		{SpecificationSource: openAPIFile, ServiceConfig: "-invalid-file-name-"},
		{SpecificationSource: secretManagerYamlFullPath, ServiceConfig: secretManagerYamlFullPath},
	} {
		if got, err := ParseOpenAPI(cfg); err == nil {
			t.Fatalf("expected error with missing source file, got=%v", got)
		}
	}
}

const openAPISingleMessagePreamble = `
{
  "openapi": "3.0.3",
  "info": {
    "title": "Secret Manager API",
    "description": "Stores sensitive data such as API keys, passwords, and certificates. Provides convenience while improving security.",
    "version": "v1"
  },
  "servers": [
    {
      "url": "https://secretmanager.googleapis.com",
      "description": "Global Endpoint"
    }
  ],
  "components": {
    "schemas": {
`

const openAPISingleMessageTrailer = `
    },
  },
  "externalDocs": {
    "description": "Find more info here.",
    "url": "https://cloud.google.com/secret-manager/"
  }
}
`
