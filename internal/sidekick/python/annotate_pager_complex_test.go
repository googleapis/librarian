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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotatePagers_MultiplePagedMethods(t *testing.T) {
	t.Parallel()

	secretMsg := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 10)
	secretsField := api.NewTestField("secrets").
		WithMessageType(secretMsg).
		WithRepeated()
	customNextToken := api.NewTestField("custom_next_token").
		WithType(api.TypezString)
	listSecretsResp := api.NewTestMessage("ListSecretsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 10).
		WithFields(secretsField, customNextToken).
		WithPagination(customNextToken, secretsField)
	customPageToken := api.NewTestField("custom_page_token").
		WithType(api.TypezString)
	listSecretsReq := api.NewTestMessage("ListSecretsRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 20).
		WithFields(customPageToken)
	listSecretsMethod := api.NewTestMethod("ListSecrets").
		WithInput(listSecretsReq).
		WithOutput(listSecretsResp).
		WithPagination(customPageToken)

	locMsg := api.NewTestMessage("Location").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/locations.proto", 10)
	locationsField := api.NewTestField("locations").
		WithMessageType(locMsg).
		WithRepeated()
	nextPageToken := api.NewTestField("next_page_token").
		WithType(api.TypezString)
	listLocationsResp := api.NewTestMessage("ListLocationsResponse").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/locations.proto", 20).
		WithFields(locationsField, nextPageToken).
		WithPagination(nextPageToken, locationsField)
	pageToken := api.NewTestField("page_token").
		WithType(api.TypezString)
	listLocationsReq := api.NewTestMessage("ListLocationsRequest").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/locations.proto", 30).
		WithFields(pageToken)
	listLocationsMethod := api.NewTestMethod("ListLocations").
		WithInput(listLocationsReq).
		WithOutput(listLocationsResp).
		WithPagination(pageToken)

	svc := api.NewTestService("SecretManagerService").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/service.proto", 40).
		WithMethods(listSecretsMethod, listLocationsMethod)
	model := api.NewTestAPI([]*api.Message{secretMsg, listSecretsReq, listSecretsResp, locMsg, listLocationsReq, listLocationsResp}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.secretmanager.v1")
	c := newTestCodec(t, model, nil)

	got := c.annotatePagers(svc)
	want := &pagersAnnotation{
		Pagers: []*pagerAnnotations{
			{
				MethodNameSnake:    "list_secrets",
				MethodNamePascal:   "ListSecrets",
				RequestType:        "service.ListSecretsRequest",
				RequestDocType:     "google.cloud.secretmanager_v1.types.ListSecretsRequest",
				ResponseType:       "service.ListSecretsResponse",
				ResponseDocType:    "google.cloud.secretmanager_v1.types.ListSecretsResponse",
				ItemType:           "resources.Secret",
				PageTokenField:     "custom_page_token",
				NextPageTokenField: "custom_next_token",
				PageableItemField:  "secrets",
			},
			{
				MethodNameSnake:    "list_locations",
				MethodNamePascal:   "ListLocations",
				RequestType:        "locations.ListLocationsRequest",
				RequestDocType:     "google.cloud.secretmanager_v1.types.ListLocationsRequest",
				ResponseType:       "locations.ListLocationsResponse",
				ResponseDocType:    "google.cloud.secretmanager_v1.types.ListLocationsResponse",
				ItemType:           "locations.Location",
				PageTokenField:     "page_token",
				NextPageTokenField: "next_page_token",
				PageableItemField:  "locations",
			},
		},
		TypeImports: []*pagerTypeImport{
			{
				Package: "google.cloud.secretmanager_v1.types",
				Module:  "locations",
			},
			{
				Package: "google.cloud.secretmanager_v1.types",
				Module:  "resources",
			},
			{
				Package: "google.cloud.secretmanager_v1.types",
				Module:  "service",
			},
		},
		HasPagers: true,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
