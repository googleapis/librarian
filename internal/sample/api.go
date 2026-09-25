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

package sample

import (
	"net/http"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	// APIName is the name of the sample API.
	APIName = "secretmanager"
	// APITitle is the title of the sample API.
	APITitle = "Secret Manager API"
	// APIPackageName is the package name of the sample API.
	APIPackageName = "google.cloud.secretmanager.v1"
	// APIDescription is the description of the sample API.
	APIDescription = "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security."
	// SpecificationName is the specification name of the sample API.
	SpecificationName = "google.cloud.secretmanager.v1"

	// ServiceName is the name of the sample service.
	ServiceName = "SecretManagerService"
	// DefaultHost is the default host of the sample service.
	DefaultHost = "secretmanager.googleapis.com"
	// Package is the package of the sample API.
	Package = "google.cloud.secretmanager.v1"
)

// API returns a sample API.
func API() *api.API {
	return api.NewTestAPI(
		[]*api.Message{
			Replication(),
			Automatic(),
		},
		[]*api.Enum{EnumState()},
		[]*api.Service{Service()},
	).
		WithName(APIName).
		WithTitle(APITitle).
		WithPackageName(APIPackageName).
		WithDescription(APIDescription)
}

// Service returns a sample service.
func Service() *api.Service {
	return api.NewTestService(ServiceName).
		WithPackage(Package).
		WithDocumentation(APIDescription).
		WithDefaultHost(DefaultHost).
		WithMethods(
			MethodCreate(),
			MethodUpdate(),
			MethodListSecretVersions(),
		)
}

// MethodCreate returns a sample create method.
func MethodCreate() *api.Method {
	m := api.NewTestMethod("CreateSecret").
		WithID("..Service.CreateSecret").
		WithDocumentation("Creates a new Secret containing no SecretVersions.").
		WithVerb(http.MethodPost).
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("secrets")).
		WithQueryParameters(map[string]bool{"secretId": true}).
		WithBodyFieldPath("body")
	// Note: InputType and OutputType are intentionally left nil because OpenAPI parser tests
	// (e.g. TestOpenAPI_MakeAPI) assert against this fixture before CrossReference runs.
	m.InputTypeID = CreateRequest().ID
	m.OutputTypeID = Secret().ID
	return m
}

// MethodUpdate returns a sample update method.
func MethodUpdate() *api.Method {
	m := api.NewTestMethod("UpdateSecret").
		WithID("..Service.UpdateSecret").
		WithDocumentation("Updates metadata of an existing Secret.").
		WithInput(UpdateRequest()).
		WithVerb(http.MethodPatch).
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariableNamed("secret", "name")).
		WithQueryParameters(map[string]bool{
			"field_mask": true,
		})
	m.OutputTypeID = ".google.protobuf.Empty"
	return m
}

// MethodAddSecretVersion returns a sample add secret version method.
func MethodAddSecretVersion() *api.Method {
	m := api.NewTestMethod("AddSecretVersion").
		WithID("..Service.AddSecretVersion").
		WithDocumentation("Creates a new SecretVersion containing secret data and attaches\nit to an existing Secret.").
		WithVerb(http.MethodPost).
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("secrets").
			WithVariableNamed("secret").
			WithVerb("addVersion")).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("body")
	m.InputTypeID = "..Service.AddSecretVersionRequest"
	m.OutputTypeID = "..SecretVersion"
	return m
}

// MethodListSecretVersions returns a sample list secret versions method.
func MethodListSecretVersions() *api.Method {
	return api.NewTestMethod("ListSecretVersions").
		WithID("..Service.ListVersion").
		WithDocumentation("Lists [SecretVersions][google.cloud.secretmanager.v1.SecretVersion]. This call does not return secret data.").
		WithInput(ListSecretVersionsRequest()).
		WithOutput(ListSecretVersionsResponse()).
		WithVerb(http.MethodPost).
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("parent").
			WithLiteral("secrets").
			WithVariableNamed("secret").
			WithVerb("listSecretVersions")).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
}

// CreateRequest returns a sample create request.
func CreateRequest() *api.Message {
	return api.NewTestMessage("CreateSecretRequest").
		WithPackage(Package).
		WithID("..Service.CreateSecretRequest").
		WithDocumentation("Request message for SecretManagerService.CreateSecret").
		WithFields(
			api.NewTestField("project").
				WithType(api.TypezString),
			api.NewTestField("secret_id").
				WithType(api.TypezString),
		)
}

// UpdateRequest returns a sample update request.
func UpdateRequest() *api.Message {
	return api.NewTestMessage("UpdateSecretRequest").
		WithPackage(Package).
		WithID("..UpdateRequest").
		WithDocumentation("Request message for SecretManagerService.UpdateSecret").
		WithFields(
			api.NewTestField("secret").
				WithType(api.TypezMessage).
				WithTypezID(Secret().ID),
			api.NewTestField("field_mask").
				WithType(api.TypezMessage).
				WithTypezID(".google.protobuf.FieldMask").
				WithOptional(),
		)
}

// ListSecretVersionsRequest returns a sample list secret versions request.
func ListSecretVersionsRequest() *api.Message {
	return api.NewTestMessage("ListSecretVersionRequest").
		WithPackage(Package).
		WithID("..ListSecretVersionsRequest").
		WithDocumentation("Lists SecretVersions. This call does not return secret data.").
		WithFields(
			api.NewTestField("parent").
				WithType(api.TypezMessage).
				WithTypezID(Secret().ID),
		)
}

// ListSecretVersionsResponse returns a sample list secret versions response.
func ListSecretVersionsResponse() *api.Message {
	return api.NewTestMessage("ListSecretVersionsResponse").
		WithPackage(Package).
		WithID("..ListSecretVersionsResponse").
		WithFields(
			api.NewTestField("versions").
				WithType(api.TypezMessage).
				WithTypezID(SecretVersion().ID).
				WithRepeated(),
		)
}

// Secret returns a sample secret.
func Secret() *api.Message {
	return api.NewTestMessage("Secret").
		WithPackage(Package).
		WithID("..Secret").
		WithFields(
			api.NewTestField("name").
				WithType(api.TypezString),
			api.NewTestField("replication").
				WithType(api.TypezMessage).
				WithTypezID(Replication().ID),
		)
}

// SecretVersion returns a sample secret version.
func SecretVersion() *api.Message {
	return api.NewTestMessage("SecretVersion").
		WithPackage(Package).
		WithID("google.cloud.secretmanager.v1.SecretVersion").
		WithEnums(EnumState()).
		WithFields(
			api.NewTestField("name").
				WithType(api.TypezString),
			api.NewTestField("state").
				WithType(api.TypezEnum).
				WithTypezID(EnumState().ID),
		)
}

// EnumState returns a sample enum state.
func EnumState() *api.Enum {
	var (
		stateEnabled = &api.EnumValue{
			Name:   "Enabled",
			Number: 1,
		}
		stateDisabled = &api.EnumValue{
			Name:   "Disabled",
			Number: 2,
		}
	)
	return &api.Enum{
		Name:    "State",
		ID:      ".test.EnumState",
		Package: Package,
		Values: []*api.EnumValue{
			stateEnabled,
			stateDisabled,
		},
	}
}

// Replication returns a sample replication.
func Replication() *api.Message {
	return &api.Message{
		Name:    "Replication",
		Package: Package,
		ID:      "google.cloud.secretmanager.v1.Replication",
		Fields: []*api.Field{
			{
				Name:     "automatic",
				Typez:    api.TypezMessage,
				TypezID:  "..Automatic",
				Optional: true,
				Repeated: false,
			},
		},
	}
}

// Automatic returns a sample automatic.
func Automatic() *api.Message {
	return &api.Message{
		Name:          "Automatic",
		ID:            "..Automatic",
		Package:       Package,
		Documentation: "A replication policy that replicates the Secret payload without any restrictions.",
		Parent:        Replication(),
		Fields: []*api.Field{
			{
				Name:          "customerManagedEncryption",
				ID:            "..Automatic.customerManagedEncryption",
				JSONName:      "customerManagedEncryption",
				Documentation: "Optional. The customer-managed encryption configuration of the Secret.",
				Typez:         api.TypezMessage,
				TypezID:       "..CustomerManagedEncryption",
				Optional:      true,
			},
		},
	}
}

// CustomerManagedEncryption returns a sample customer managed encryption.
func CustomerManagedEncryption() *api.Message {
	return &api.Message{
		Name:    "CustomerManagedEncryption",
		ID:      "..CustomerManagedEncryption",
		Package: Package,
	}
}

// SecretPayload returns a sample secret payload.
func SecretPayload() *api.Message {
	return &api.Message{
		Name:          "SecretPayload",
		ID:            "..SecretPayload",
		Documentation: "A secret payload resource in the Secret Manager API. This contains the\nsensitive secret payload that is associated with a SecretVersion.",
		Fields: []*api.Field{
			{
				Name:          "data",
				ID:            "..SecretPayload.data",
				JSONName:      "data",
				Documentation: "The secret data. Must be no larger than 64KiB.",
				Typez:         api.TypezBytes,
				TypezID:       "bytes",
				Optional:      true,
			},
			{
				Name:          "dataCrc32c",
				ID:            "..SecretPayload.dataCrc32c",
				JSONName:      "dataCrc32c",
				Documentation: "Optional. If specified, SecretManagerService will verify the integrity of the\nreceived data on SecretManagerService.AddSecretVersion calls using\nthe crc32c checksum and store it to include in future\nSecretManagerService.AccessSecretVersion responses. If a checksum is\nnot provided in the SecretManagerService.AddSecretVersion request, the\nSecretManagerService will generate and store one for you.\n\nThe CRC32C value is encoded as a Int64 for compatibility, and can be\nsafely downconverted to uint32 in languages that support this type.\nhttps://cloud.google.com/apis/design/design_patterns#integer_types",
				Typez:         api.TypezInt64,
				TypezID:       "int64",
				Optional:      true,
			},
		},
	}
}
