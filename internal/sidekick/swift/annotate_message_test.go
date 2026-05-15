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
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMessage(t *testing.T) {
	for _, test := range []struct {
		name    string
		message *api.Message
		want    *messageAnnotations
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
			},
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
			},
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
			},
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
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, f := range test.message.Fields {
				f.Parent = test.message
			}
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, test.message.Codec, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
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
		Name:    "ListSecretsRequest",
		TypeURL: "type.googleapis.com/google.cloud.secretmanager.v1.ListSecretsRequest",
	}
	if diff := cmp.Diff(wantRequest, gotRequest, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	// Verify annotations on response message
	gotResponse := outputType.Codec.(*messageAnnotations)
	wantResponse := &messageAnnotations{
		Name:                "ListSecretsResponse",
		TypeURL:             "type.googleapis.com/google.cloud.secretmanager.v1.ListSecretsResponse",
		IsPaginatedResponse: true,
		PageableItemField:   "secrets",
		PageableItemType:    "Secret",
		ImportsGax:          true,
	}
	if diff := cmp.Diff(wantResponse, gotResponse, cmpopts.IgnoreFields(messageAnnotations{}, "Model")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}
