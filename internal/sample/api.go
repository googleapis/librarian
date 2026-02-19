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

// SecretManagerAPI returns a sample Secret Manager API.
func SecretManagerAPI() *api.API {
	svc := Service()
	messages := []*api.Message{
		Replication(),
		Automatic(),
		CustomerManagedEncryption(),
		SecretPayload(),
		Secret(),
		SecretVersion(),
		CreateRequest(),
		UpdateRequest(),
		ListSecretVersionsRequest(),
		ListSecretVersionsResponse(),
	}
	enums := []*api.Enum{EnumState()}

	model := api.NewTestAPI(messages, enums, []*api.Service{svc})
	model.Name = APIName
	model.Title = APITitle
	model.PackageName = APIPackageName
	model.Description = APIDescription

	_ = api.CrossReference(model)
	return model
}

// VisionAIAPI returns a sample Vision AI API.
func VisionAIAPI() *api.API {
	svc := &api.Service{
		Name:          "AppPlatform",
		Documentation: "Service describing handlers for resources",
		DefaultHost:   "visionai.googleapis.com",
		Package:       "google.cloud.visionai.v1",
		Methods: []*api.Method{
			MethodCreateApplicationLRO(),
			MethodUpdateApplicationLRO(),
			MethodDeleteApplicationLRO(),
		},
	}

	messages := []*api.Message{
		CreateApplicationRequest(),
		UpdateApplicationRequest(),
		DeleteApplicationRequest(),
		Application(),
		Operation(),
	}

	model := api.NewTestAPI(messages, nil, []*api.Service{svc})
	model.Name = "visionai"
	model.Title = "Vision AI API"
	model.PackageName = "google.cloud.visionai.v1"
	model.Description = "Vision AI App Platform"

	_ = api.CrossReference(model)
	return model
}

// Operation returns a sample LRO Operation message.
func Operation() *api.Message {
	return &api.Message{
		Name:    "Operation",
		ID:      ".google.longrunning.Operation",
		Package: "google.longrunning",
		Fields: []*api.Field{
			{
				Name:  "name",
				Typez: api.STRING_TYPE,
			},
			{
				Name:  "done",
				Typez: api.BOOL_TYPE,
			},
		},
	}
}

// Method returns a fully wired method from the standard Secret Manager API sample.
func Method(id string) *api.Method {
	model := SecretManagerAPI()
	return model.State.MethodByID[id]
}

// VisionAIMethod returns a fully wired method from the Vision AI API sample.
func VisionAIMethod(id string) *api.Method {
	model := VisionAIAPI()
	return model.State.MethodByID[id]
}

// API returns the default sample API (Secret Manager) for backwards compatibility.
func API() *api.API {
	return SecretManagerAPI()
}

// Service returns a sample service.
func Service() *api.Service {
	return &api.Service{
		Name:          ServiceName,
		Documentation: APIDescription,
		DefaultHost:   DefaultHost,
		Methods: []*api.Method{
			MethodCreate(),
			MethodUpdate(),
			MethodListSecretVersions(),
		},
		Package: Package,
	}
}

// withSecretManagerModel wires the method to the fully resolved SecretManagerAPI model.
func withSecretManagerModel(methodID string, fallback *api.Method) *api.Method {
	if m, ok := SecretManagerAPI().State.MethodByID[methodID]; ok {
		return m
	}
	return fallback
}

// withVisionAIModel wires the method to the fully resolved VisionAIAPI model.
func withVisionAIModel(methodID string, fallback *api.Method) *api.Method {
	if m, ok := VisionAIAPI().State.MethodByID[methodID]; ok {
		return m
	}
	return fallback
}

// MethodCreate returns a sample create method.
func MethodCreate() *api.Method {
	return &api.Method{
		Name:          "CreateSecret",
		Documentation: "Creates a new Secret containing no SecretVersions.",
		ID:            "..Service.CreateSecret",
		InputTypeID:   CreateRequest().ID,
		OutputTypeID:  Secret().ID,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodPost,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("secrets"),
					QueryParameters: map[string]bool{"secretId": true},
				},
			},
			BodyFieldPath: "body",
		},
	}
}

// MethodUpdate returns a sample update method.
func MethodUpdate() *api.Method {
	return &api.Method{
		Name:          "UpdateSecret",
		Documentation: "Updates metadata of an existing Secret.",
		ID:            "..Service.UpdateSecret",
		InputTypeID:   UpdateRequest().ID,
		OutputTypeID:  ".google.protobuf.Empty",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodPatch,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithVariableNamed("secret", "name"),
					QueryParameters: map[string]bool{
						"field_mask": true,
					},
				},
			},
		},
	}
}

// MethodAddSecretVersion returns a sample add secret version method.
func MethodAddSecretVersion() *api.Method {
	return &api.Method{
		Name:          "AddSecretVersion",
		ID:            "..Service.AddSecretVersion",
		Documentation: "Creates a new SecretVersion containing secret data and attaches\nit to an existing Secret.",
		InputTypeID:   "..Service.AddSecretVersionRequest",
		OutputTypeID:  "..SecretVersion",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodPost,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("secrets").
						WithVariableNamed("secret").
						WithVerb("addVersion"),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "body",
		},
	}
}

// MethodListSecretVersions returns a sample list secret versions method.
func MethodListSecretVersions() *api.Method {
	return &api.Method{
		Name:          "ListSecretVersions",
		ID:            "..Service.ListVersion",
		Documentation: "Lists [SecretVersions][google.cloud.secretmanager.v1.SecretVersion]. This call does not return secret data.",
		InputTypeID:   ListSecretVersionsRequest().ID,
		InputType:     ListSecretVersionsRequest(),
		OutputTypeID:  ListSecretVersionsResponse().ID,
		OutputType:    ListSecretVersionsResponse(),
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodGet,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("parent").
						WithLiteral("secrets").
						WithVariableNamed("secret").
						WithVerb("listSecretVersions"),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "*",
		},
	}
}

// CreateRequest returns a sample create request.
func CreateRequest() *api.Message {
	return &api.Message{
		Name:          "CreateSecretRequest",
		ID:            "..Service.CreateSecretRequest",
		Documentation: "Request message for SecretManagerService.CreateSecret",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:     "project",
				JSONName: "project",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "secret_id",
				JSONName: "secretId",
				Typez:    api.STRING_TYPE,
			},
		},
	}
}

// UpdateRequest returns a sample update request.
func UpdateRequest() *api.Message {
	return &api.Message{
		Name:          "UpdateSecretRequest",
		ID:            "..UpdateRequest",
		Documentation: "Request message for SecretManagerService.UpdateSecret",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:     "secret",
				JSONName: "secret",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  Secret().ID,
			},
			{
				Name:     "field_mask",
				JSONName: "fieldMask",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".google.protobuf.FieldMask",
				Optional: true,
			},
		},
	}
}

// ListSecretVersionsRequest returns a sample list secret versions request.
func ListSecretVersionsRequest() *api.Message {
	return &api.Message{
		Name:          "ListSecretVersionRequest",
		ID:            "..ListSecretVersionsRequest",
		Documentation: "Lists SecretVersions. This call does not return secret data.",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				ID:       Secret().ID + ".parent",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  Secret().ID,
			},
		},
	}
}

// ListSecretVersionsResponse returns a sample list secret versions response.
func ListSecretVersionsResponse() *api.Message {
	return &api.Message{
		Name:    "ListSecretVersionsResponse",
		ID:      "..ListSecretVersionsResponse",
		Package: Package,
		Fields: []*api.Field{
			{
				Name:        "versions",
				JSONName:    "versions",
				Typez:       api.MESSAGE_TYPE,
				TypezID:     SecretVersion().ID,
				MessageType: SecretVersion(),
				Repeated:    true,
			},
		},
	}
}

// Secret returns a sample secret.
func Secret() *api.Message {
	return &api.Message{
		Name:    "Secret",
		ID:      "..Secret",
		Package: Package,
		Fields: []*api.Field{
			{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "replication",
				JSONName: "replication",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  Replication().ID,
			},
		},
	}
}

// SecretVersion returns a sample secret version.
func SecretVersion() *api.Message {
	return &api.Message{
		Name:    "SecretVersion",
		Package: Package,
		ID:      "google.cloud.secretmanager.v1.SecretVersion",
		Enums:   []*api.Enum{EnumState()},
		Fields: []*api.Field{
			{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:     "state",
				JSONName: "state",
				Typez:    api.ENUM_TYPE,
				TypezID:  EnumState().ID,
			},
		},
	}
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
				Typez:    api.MESSAGE_TYPE,
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
				JSONName:      "customerManagedEncryption",
				Documentation: "Optional. The customer-managed encryption configuration of the Secret.",
				Typez:         api.MESSAGE_TYPE,
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
				JSONName:      "data",
				Documentation: "The secret data. Must be no larger than 64KiB.",
				Typez:         api.BYTES_TYPE,
				TypezID:       "bytes",
				Optional:      true,
			},
			{
				Name:          "dataCrc32c",
				JSONName:      "dataCrc32c",
				Documentation: "Optional. If specified, SecretManagerService will verify the integrity of the\nreceived data on SecretManagerService.AddSecretVersion calls using\nthe crc32c checksum and store it to include in future\nSecretManagerService.AccessSecretVersion responses. If a checksum is\nnot provided in the SecretManagerService.AddSecretVersion request, the\nSecretManagerService will generate and store one for you.\n\nThe CRC32C value is encoded as a Int64 for compatibility, and can be\nsafely downconverted to uint32 in languages that support this type.\nhttps://cloud.google.com/apis/design/design_patterns#integer_types",
				Typez:         api.INT64_TYPE,
				TypezID:       "int64",
				Optional:      true,
			},
		},
	}
}

// MethodCreateApplicationLRO returns a sample create method that returns a long-running operation.
func MethodCreateApplicationLRO() *api.Method {
	return &api.Method{
		Name:          "CreateApplication",
		Documentation: "Creates a new Application in a given project and location.",
		ID:            "..Service.CreateApplication",
		InputTypeID:   "..Service.CreateApplicationRequest",
		InputType:     CreateApplicationRequest(),
		OutputTypeID:  ".google.longrunning.Operation",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodPost,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("parent").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()).
						WithLiteral("applications"),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "application",
		},
		OperationInfo: &api.OperationInfo{
			ResponseTypeID: Application().ID,
			MetadataTypeID: "..OperationMetadata",
		},
	}
}

// MethodUpdateApplicationLRO returns a sample update method that returns a long-running operation.
func MethodUpdateApplicationLRO() *api.Method {
	return &api.Method{
		Name:          "UpdateApplication",
		Documentation: "Updates the parameters of a single Application.",
		ID:            "..Service.UpdateApplication",
		InputTypeID:   "..Service.UpdateApplicationRequest",
		InputType:     UpdateApplicationRequest(),
		OutputTypeID:  ".google.longrunning.Operation",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodPatch,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("application", "name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("applications").WithMatch()),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "application",
		},
		OperationInfo: &api.OperationInfo{
			ResponseTypeID: Application().ID,
			MetadataTypeID: "..OperationMetadata",
		},
	}
}

// MethodDeleteApplicationLRO returns a sample delete method that returns a long-running operation.
func MethodDeleteApplicationLRO() *api.Method {
	return &api.Method{
		Name:          "DeleteApplication",
		Documentation: "Deletes a single Application.",
		ID:            "..Service.DeleteApplication",
		InputTypeID:   "..Service.DeleteApplicationRequest",
		InputType:     DeleteApplicationRequest(),
		OutputTypeID:  ".google.longrunning.Operation",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: http.MethodDelete,
					PathTemplate: api.NewPathTemplate().
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("applications").WithMatch()),
					QueryParameters: map[string]bool{},
				},
			},
		},
		OperationInfo: &api.OperationInfo{
			ResponseTypeID: ".google.protobuf.Empty",
			MetadataTypeID: "..OperationMetadata",
		},
	}
}

// CreateApplicationRequest returns a sample create application request.
func CreateApplicationRequest() *api.Message {
	return &api.Message{
		Name:          "CreateApplicationRequest",
		ID:            "..Service.CreateApplicationRequest",
		Documentation: "Request message for CreateApplication",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:     "parent",
				JSONName: "parent",
				Typez:    api.STRING_TYPE,
			},
			{
				Name:        "application",
				JSONName:    "application",
				Typez:       api.MESSAGE_TYPE,
				TypezID:     Application().ID,
				MessageType: Application(),
			},
		},
	}
}

// UpdateApplicationRequest returns a sample update application request.
func UpdateApplicationRequest() *api.Message {
	return &api.Message{
		Name:          "UpdateApplicationRequest",
		ID:            "..Service.UpdateApplicationRequest",
		Documentation: "Request message for UpdateApplication",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:        "application",
				JSONName:    "application",
				Typez:       api.MESSAGE_TYPE,
				TypezID:     Application().ID,
				MessageType: Application(),
			},
			{
				Name:     "update_mask",
				JSONName: "updateMask",
				Typez:    api.MESSAGE_TYPE,
				TypezID:  ".google.protobuf.FieldMask",
				Optional: true,
			},
		},
	}
}

// DeleteApplicationRequest returns a sample delete application request.
func DeleteApplicationRequest() *api.Message {
	return &api.Message{
		Name:          "DeleteApplicationRequest",
		ID:            "..Service.DeleteApplicationRequest",
		Documentation: "Request message for DeleteApplication",
		Package:       Package,
		Fields: []*api.Field{
			{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
			},
		},
	}
}

// Application returns a sample application.
func Application() *api.Message {
	return &api.Message{
		Name:    "Application",
		ID:      "..Application",
		Package: Package,
		Resource: &api.Resource{
			Type:     "example.googleapis.com/Application",
			Plural:   "applications",
			Singular: "application",
			Patterns: []api.ResourcePattern{
				{
					*api.NewPathSegment().WithLiteral("projects"),
					*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
					*api.NewPathSegment().WithLiteral("locations"),
					*api.NewPathSegment().WithVariable(api.NewPathVariable("location").WithMatch()),
					*api.NewPathSegment().WithLiteral("applications"),
					*api.NewPathSegment().WithVariable(api.NewPathVariable("application").WithMatch()),
				},
			},
		},
		Fields: []*api.Field{
			{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
			},
		},
	}
}
