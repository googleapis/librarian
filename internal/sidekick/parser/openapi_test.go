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
	wantLocation := api.NewTestMessage("Location").
		WithPackage("").
		WithID("..Location").
		WithDocumentation("A resource that represents a Google Cloud location.").
		WithFields(
			api.NewTestField("name").
				WithDocumentation("Resource name for the location, which may vary between implementations."+"\nFor example: `\"projects/example-project/locations/us-east1\"`").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("locationId").
				WithDocumentation("The canonical id for this location. For example: `\"us-east1\"`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("displayName").
				WithDocumentation(`The friendly name for this location, typically a nearby city name.`+"\n"+`For example, "Tokyo".`).
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("labels").
				WithDocumentation("Cross-service attributes for the location. For example\n\n    {\"cloud.googleapis.com/region\": \"us-east1\"}").
				WithType(api.TypezMessage).
				WithTypezID("$map<string, string>").
				WithMap(),
			api.NewTestField("metadata").
				WithDocumentation(`Service-specific metadata. For example the available capacity at the given`+"\n"+`location.`).
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.Any").
				WithOptional(),
		)
	apitest.CheckMessage(t, location, wantLocation)

	listLocationsResponse := test.Message("..ListLocationsResponse")
	if listLocationsResponse == nil {
		t.Errorf("missing message (ListLocationsResponse) in MessageByID index")
		return
	}
	nextPageToken := api.NewTestField("nextPageToken").
		WithDocumentation("The standard List next-page token.").
		WithType(api.TypezString).
		WithTypezID("string").
		WithOptional()
	locations := api.NewTestField("locations").
		WithDocumentation("A list of locations that matches the specified filter in the request.").
		WithType(api.TypezMessage).
		WithTypezID("..Location").
		WithRepeated()
	wantListLocationsResponse := api.NewTestMessage("ListLocationsResponse").
		WithPackage("").
		WithID("..ListLocationsResponse").
		WithDocumentation("The response message for Locations.ListLocations.").
		WithPagination(nextPageToken, locations)
	apitest.CheckMessage(t, listLocationsResponse, wantListLocationsResponse)

	// This is a synthetic message, the OpenAPI spec does not contain requests
	// messages for messages without a body.
	pageToken := api.NewTestField("pageToken").
		WithDocumentation("A page token received from the `next_page_token` field in the response.\nSend that page token to receive the subsequent page.").
		WithType(api.TypezString).
		WithTypezID("string").
		WithOptional()
	want := api.NewTestMessage("ListLocationsRequest").
		WithPackage("").
		WithID("..Service.ListLocationsRequest").
		WithDocumentation("Synthetic request message for the [ListLocations()][.Service.ListLocations] method.").
		WithFields(
			api.NewTestField("project").
				WithDocumentation("The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/locations`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("filter").
				WithDocumentation("A filter to narrow down results to a preferred subset."+
					"\nThe filtering language accepts strings like `\"displayName=tokyo"+
					"\"`, and\nis documented in more detail in [AIP-160](https://google"+
					".aip.dev/160).").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("pageSize").
				WithDocumentation("The maximum number of results to return.\nIf not set, the service selects a default.").
				WithType(api.TypezInt32).
				WithTypezID("int32").
				WithOptional(),
			pageToken,
		)
	want.SyntheticRequest = true
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

	wantListLocationsMethod := api.NewTestMethod("ListLocations").
		WithID("..Service.ListLocations").
		WithDocumentation("Lists information about the supported locations for this service.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("locations")).
		WithQueryParameters(map[string]bool{
			"filter":    true,
			"pageSize":  true,
			"pageToken": true,
		}).
		WithPagination(pageToken)
	wantListLocationsMethod.IsList = false
	wantListLocationsMethod.InputTypeID = "..Service.ListLocationsRequest"
	wantListLocationsMethod.OutputTypeID = "..ListLocationsResponse"
	apitest.CheckMethod(t, service, "ListLocations", wantListLocationsMethod)

	cs := sample.MethodCreate()
	apitest.CheckMethod(t, service, cs.Name, cs)

	asv := sample.MethodAddSecretVersion()
	apitest.CheckMethod(t, service, asv.Name, asv)
}

func TestOpenAPI_ServicePlaceholder(t *testing.T) {
	test := openapiSecretManagerAPI(t)
	want := api.NewTestMessage("Service").
		WithPackage("").
		WithID("..Service").
		WithDocumentation("Synthetic messages for the [Service][.Service] service.")
	want.ServicePlaceholder = true
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
	listFoosMethod := api.NewTestMethod("ListFoos").
		WithID("..Service.ListFoos").
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("foos"),
		).
		WithQueryParameters(map[string]bool{"pageSize": true, "pageToken": true}).
		WithPagination(
			api.NewTestField("pageToken").
				WithDocumentation("The `{pageToken}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/foos`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
		)
	listFoosMethod.Pagination.ID = "..Service.ListFoosRequest.pageToken"
	listFoosMethod.IsList = false
	listFoosMethod.InputTypeID = "..Service.ListFoosRequest"
	listFoosMethod.OutputTypeID = "..ListFoosResponse"

	wantService := api.NewTestService("Service").
		WithPackage("").
		WithMethods(listFoosMethod)
	apitest.CheckService(t, service, wantService)
	resp := test.Message("..ListFoosResponse")
	if resp == nil {
		t.Errorf("missing message (ListFoosResponse) in MessageByID index")
		return
	}
	wantResp := api.NewTestMessage("ListFoosResponse").
		WithPackage("").
		WithID("..ListFoosResponse").
		WithPagination(
			api.NewTestField("nextPageToken").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("secrets").
				WithType(api.TypezMessage).
				WithTypezID("..Foo").
				WithRepeated(),
		)
	apitest.CheckMessage(t, resp, wantResp)
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

	requestID := api.NewTestField("requestId").
		WithDocumentation("Test-only Description").
		WithType(api.TypezString).
		WithTypezID("string").
		WithOptional().
		WithAutoPopulated()
	requestIDExplicit := api.NewTestField("requestIdExplicitlyNotRequired").
		WithDocumentation("Test-only Description").
		WithType(api.TypezString).
		WithTypezID("string").
		WithOptional().
		WithAutoPopulated()
	wantMessage := api.NewTestMessage("CreateFooRequest").
		WithPackage("test").
		WithID(".test.TestService.CreateFooRequest").
		WithDocumentation("Synthetic request message for the [CreateFoo()][test.TestService.CreateFoo] method.").
		WithFields(
			api.NewTestField("project").
				WithDocumentation("The `{project}` component of the target path.\n\nThe full target path will be in the form `/v1/projects/{project}/foos`.").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("fooId").
				WithDocumentation("Test-only Description").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			requestID,
			requestIDExplicit,
			api.NewTestField("notRequestIdRequired").
				WithDocumentation("Test-only Description").
				WithType(api.TypezString).
				WithTypezID("string").
				WithBehavior(api.FieldBehaviorRequired),
			api.NewTestField("notRequestIdMissingFormat").
				WithDocumentation("Test-only Description").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("notRequestIdMissingServiceConfig").
				WithDocumentation("Test-only Description").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional().
				WithAutoPopulated(),
		)
	wantMessage.SyntheticRequest = true
	message := test.Message(wantMessage.ID)
	if message == nil {
		t.Fatalf("Cannot find message %s in API State", wantMessage.ID)
	}
	apitest.CheckMessage(t, message, wantMessage)

	method := test.Method(".test.TestService.CreateFoo")
	if method == nil {
		t.Fatalf("Cannot find method %s in API State", ".test.TestService.CreateFoo")
	}
	wantField := []*api.Field{requestID, requestIDExplicit}
	if diff := cmp.Diff(wantField, method.AutoPopulated, cmpopts.IgnoreFields(api.Field{}, "Parent")); diff != "" {
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
	rpcA := api.NewTestMethod("RpcA").
		WithID("..Service.RpcA").
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("rpc").
				WithLiteral("a"),
		).
		WithQueryParameters(map[string]bool{"filter": true})
	rpcA.InputTypeID = "..Service.RpcARequest"
	rpcA.OutputTypeID = "..Response"
	apitest.CheckMethod(t, service, "RpcA", rpcA)

	rpcB := api.NewTestMethod("RpcB").
		WithID("..Service.RpcB").
		WithDeprecated(true).
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("rpc").
				WithLiteral("b"),
		).
		WithQueryParameters(map[string]bool{})
	rpcB.InputTypeID = "..Service.RpcBRequest"
	rpcB.OutputTypeID = "..Response"
	apitest.CheckMethod(t, service, "RpcB", rpcB)

	response := test.Message("..Response")
	if response == nil {
		t.Errorf("cannot find message %s", "..Response")
		return
	}
	wantResponse := api.NewTestMessage("Response").
		WithPackage("").
		WithID("..Response").
		WithFields(
			api.NewTestField("name").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
			api.NewTestField("other").
				WithType(api.TypezString).
				WithTypezID("string").
				WithDeprecated(true).
				WithOptional(),
		)
	apitest.CheckMessage(t, response, wantResponse)

	deprecatedMessage := test.Message("..DeprecatedMessage")
	if deprecatedMessage == nil {
		t.Errorf("cannot find message %s", "..DeprecatedMessage")
		return
	}
	wantDeprecatedMessage := api.NewTestMessage("DeprecatedMessage").
		WithPackage("").
		WithID("..DeprecatedMessage").
		WithDeprecated(true).
		WithFields(
			api.NewTestField("name").
				WithType(api.TypezString).
				WithTypezID("string").
				WithOptional(),
		)
	apitest.CheckMessage(t, deprecatedMessage, wantDeprecatedMessage)
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
