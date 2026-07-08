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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_MapPagination(t *testing.T) {
	for _, test := range []struct {
		name              string
		optional          bool
		wantNextPageToken string
	}{
		{
			name:     "Required",
			optional: false,
			wantNextPageToken: `public func _nextPageToken() -> Swift.String {
    return self.nextPageToken
  }`,
		},
		{
			name:     "Optional",
			optional: true,
			wantNextPageToken: `public func _nextPageToken() -> Swift.String {
    return self.nextPageToken ?? ""
  }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

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

			secretType := &api.Message{
				Name:    "Secret",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.Secret",
			}

			keyField := &api.Field{Name: "key", JSONName: "key", Typez: api.TypezString}
			valueField := &api.Field{Name: "value", JSONName: "value", Typez: api.TypezMessage, TypezID: secretType.ID, MessageType: secretType}
			mapEntryType := &api.Message{
				Name:    "SecretsEntry",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.ListSecretsResponse.SecretsEntry",
				IsMap:   true,
				Fields:  []*api.Field{keyField, valueField},
			}
			keyField.Parent = mapEntryType
			valueField.Parent = mapEntryType

			itemField := &api.Field{Name: "secrets", JSONName: "secrets", Typez: api.TypezMessage, TypezID: mapEntryType.ID, MessageType: mapEntryType, Map: true}
			nextPageTokenField := &api.Field{Name: "next_page_token", JSONName: "nextPageToken", Typez: api.TypezString, Optional: test.optional}
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

			iam := &api.Service{
				Name: "SecretManagerService",
				Methods: []*api.Method{
					{
						Name:          "ListSecrets",
						Documentation: "Lists secrets.",
						InputTypeID:   inputType.ID,
						InputType:     inputType,
						OutputTypeID:  outputType.ID,
						OutputType:    outputType,
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

			model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType, mapEntryType}, nil, []*api.Service{iam})
			model.PackageName = "google.cloud.secretmanager.v1"

			cfg := &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year": "2038",
				},
			}

			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{
					Name:               "GoogleCloudGax",
					RequiredByServices: true,
				},
				{
					Name:               "GoogleCloudAuth",
					RequiredByServices: true,
				},
			})

			if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
				t.Fatal(err)
			}

			verifyGeneratedMapService(t, outDir)
			verifyGeneratedMapResponse(t, outDir, test.wantNextPageToken)
		})
	}
}

func verifyGeneratedMapService(t *testing.T, outDir string) {
	t.Helper()
	filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotMethodOverload := extractBlock(t, contentStr, `  public func listSecrets(
    byItem: `, "\n  }")
	wantMethodOverload := `  public func listSecrets(
    byItem: ListSecretsRequest, options: GoogleCloudGax.RequestOptions
) throws -> any AsyncSequence<(Swift.String, Secret), Swift.Error>
 {
    let listRpc = { (token: String) async throws -> GoogleCloudSecretmanagerV1.ListSecretsResponse in
      var request = byItem
      request.pageToken = token
      return try await self.listSecrets(request: request, options: options)
    }
    return GoogleCloudGax.PaginatedResponseSequence(listRpc: listRpc)
  }`
	if diff := cmp.Diff(wantMethodOverload, gotMethodOverload); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func verifyGeneratedMapResponse(t *testing.T, outDir string, wantNextPageToken string) {
	t.Helper()
	respFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "ListSecretsResponse.swift")
	respContent, err := os.ReadFile(respFilename)
	if err != nil {
		t.Fatal(err)
	}
	respContentStr := string(respContent)

	gotResponseMessage := extractBlock(t, respContentStr, "public struct ListSecretsResponse: ", "{")
	for _, p := range []string{"Codable", "Equatable", "GoogleCloudWkt._AnyPackable", "GoogleCloudGax._PaginatedResponse", "Sendable"} {
		if !strings.Contains(gotResponseMessage, p) {
			t.Errorf("expected %q in ListSecretsResponse declaration, got: %s", p, gotResponseMessage)
		}
	}

	gotGetItems := extractBlock(t, respContentStr, "public func _getPaginatedItems()", "  }")
	wantGetItems := `public func _getPaginatedItems() -> [(Swift.String, Secret)] {
    return self.secrets.map { ($0, $1) }
  }`
	if diff := cmp.Diff(wantGetItems, gotGetItems); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotNextPageToken := extractBlock(t, respContentStr, "public func _nextPageToken()", "  }")
	if diff := cmp.Diff(wantNextPageToken, gotNextPageToken); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
