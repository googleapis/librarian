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
			wantNextPageToken: `  public func _nextPageToken() -> Swift.String {
    return self.nextPageToken
  }`,
		},
		{
			name:     "Optional",
			optional: true,
			wantNextPageToken: `  public func _nextPageToken() -> Swift.String {
    return self.nextPageToken ?? ""
  }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
			pageTokenField := api.NewTestField("page_token").WithType(api.TypezString)
			inputType := api.NewTestMessage("ListSecretsRequest").
				WithPackage("google.cloud.secretmanager.v1").
				WithFields(pageSizeField, pageTokenField)

			secretType := api.NewTestMessage("Secret").
				WithPackage("google.cloud.secretmanager.v1")

			outputType := api.NewTestMessage("ListSecretsResponse").
				WithPackage("google.cloud.secretmanager.v1")

			keyField := api.NewTestField("key").WithType(api.TypezString)
			valueField := api.NewTestField("value").WithMessageType(secretType)
			mapEntryType := api.NewTestMessage("SecretsEntry").
				WithFields(keyField, valueField).
				WithIsMap()
			outputType.WithMessages(mapEntryType)

			itemField := api.NewTestField("secrets").
				WithMessageType(mapEntryType).
				WithMap()
			nextPageTokenField := api.NewTestField("next_page_token").
				WithType(api.TypezString)
			if test.optional {
				nextPageTokenField.WithOptional()
			}
			outputType.
				WithFields(itemField, nextPageTokenField).
				WithPagination(nextPageTokenField, itemField)

			method := api.NewTestMethod("ListSecrets").
				WithDocumentation("Lists secrets.").
				WithInput(inputType).
				WithOutput(outputType).
				WithVerb("GET").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("secrets")).
				WithPagination(pageTokenField)

			iam := api.NewTestService("SecretManagerService").
				WithPackage("google.cloud.secretmanager.v1").
				WithMethods(method)

			model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})

			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{
					Name:               "GoogleGax",
					RequiredByServices: true,
				},
				{
					Name:               "GoogleAuth",
					RequiredByServices: true,
				},
			})

			library := &config.Library{
				Swift: swiftCfg,
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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
    byItem: ListSecretsRequest, options: GoogleGax.RequestOptions
) -> any AsyncSequence<(Swift.String, Secret), Swift.Error>
 {
    let listRpc = { (token: Swift.String) async throws -> GoogleCloudSecretmanagerV1.ListSecretsResponse in
      var request = byItem
      request.pageToken = token
      return try await self.listSecrets(request: request, options: options)
    }
    return GoogleGax.PaginatedResponseSequence(listRpc: listRpc)
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
	for _, p := range []string{"Codable", "Equatable", "GoogleWKT._AnyPackable", "Sendable"} {
		if !strings.Contains(gotResponseMessage, p) {
			t.Errorf("expected %q in ListSecretsResponse declaration, got: %s", p, gotResponseMessage)
		}
	}

	gotExtension := extractBlock(t, respContentStr, "@_spi(GoogleCloudInternal)\nextension ListSecretsResponse: GoogleGax._PaginatedResponse {", "\n}")
	wantGetItems := `  public func _getPaginatedItems() -> [(Swift.String, Secret)] {
    return self.secrets.map { ($0, $1) }
  }`
	if !strings.Contains(gotExtension, wantGetItems) {
		t.Errorf("expected %q in ListSecretsResponse extension, got:\n%s", wantGetItems, gotExtension)
	}
	if !strings.Contains(gotExtension, wantNextPageToken) {
		t.Errorf("expected %q in ListSecretsResponse extension, got:\n%s", wantNextPageToken, gotExtension)
	}
}
