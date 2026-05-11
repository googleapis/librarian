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

func TestGenerateService_Files(t *testing.T) {
	outDir := t.TempDir()

	iam := &api.Service{Name: "IAM"}
	secretManager := &api.Service{Name: "SecretManagerService"}

	model := api.NewTestAPI(nil, nil, []*api.Service{iam, secretManager})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"IAM.swift", "SecretManagerService.swift", "Clients.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateServiceSwift_SnippetReference(t *testing.T) {
	outDir := t.TempDir()

	// "Protocol" is a reserved word that gets mangled to "Protocol_"
	service := &api.Service{Name: "Protocol"}

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	// The file name uses the unmangled name
	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Protocol.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock := extractBlock(t, contentStr, "/// @Snippet", "public protocol Protocol_ {")
	wantBlock := `/// @Snippet(path: "ProtocolQuickstart")
public protocol Protocol_ {`
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_SnippetFiles(t *testing.T) {
	outDir := t.TempDir()

	dummyMessage := &api.Message{Name: "DummyMessage"}
	iam := &api.Service{
		Name: "IAM",
		Methods: []*api.Method{
			{
				Name:      "CreateRole",
				InputType: dummyMessage,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}
	secretManager := &api.Service{
		Name: "SecretManagerService",
		Methods: []*api.Method{
			{
				Name:      "GetSecret",
				InputType: dummyMessage,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{dummyMessage}, nil, []*api.Service{iam, secretManager})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Snippets")
	expectedFiles := []string{
		"IAMQuickstart.swift",
		"SecretManagerServiceQuickstart.swift",
		"IAM_CreateRole.swift",
		"SecretManagerService_GetSecret.swift",
	}
	for _, expected := range expectedFiles {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateService_WithImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	inputMessage := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.TypezMessage,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	iam := &api.Service{
		Name: "IAM",
		Methods: []*api.Method{
			{
				Name:      "TestMethod",
				InputType: inputMessage,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputMessage}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.test.v1"
	model.AddMessage(externalMessage)

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
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
	})

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "IAM.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	expectedImports := `import GoogleCloudAuth
import GoogleCloudGax

import GoogleCloudExternalV1`

	if !strings.Contains(contentStr, expectedImports) {
		t.Errorf("expected imports block not found in %s. Got content:\n%s", filename, contentStr)
	}
}

func TestGenerateService_PathParameters(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      *api.PathTemplate
		wantBlock string
	}{
		{
			name: "Nested",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("secret", "name"),
			wantBlock: `let path = try { () throws -> String in
        guard let pathVariable0 = request.secret.map({ $0.name }), !pathVariable0.isEmpty else {
          throw GoogleCloudGax.RequestError.binding("'request.secret.name' is not set or is empty")
        }
        return "/v1/\(pathVariable0)"
      }()`,
		},
		{
			name: "Plain",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("name"),
			wantBlock: `let path = try { () throws -> String in
        guard let pathVariable0 = request.name as String?, !pathVariable0.isEmpty else {
          throw GoogleCloudGax.RequestError.binding("'request.name' is not set or is empty")
        }
        return "/v1/\(pathVariable0)"
      }()`,
		},
		{
			name: "Multiple strings",
			path: (&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("locations").
				WithVariableNamed("location"),
			wantBlock: `let path = try { () throws -> String in
        guard let pathVariable0 = request.project as String?, !pathVariable0.isEmpty else {
          throw GoogleCloudGax.RequestError.binding("'request.project' is not set or is empty")
        }
        guard let pathVariable1 = request.location, !pathVariable1.isEmpty else {
          throw GoogleCloudGax.RequestError.binding("'request.location' is not set or is empty")
        }
        return "/v1/projects/\(pathVariable0)/locations/\(pathVariable1)"
      }()`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			secretMessage := &api.Message{
				Name:    "Secret",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.Secret",
				Fields: []*api.Field{
					{
						Name:  "name",
						Typez: api.TypezString,
					},
				},
			}

			requestMessage := &api.Message{
				Name:    "CreateSecretRequest",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.CreateSecretRequest",
				Fields: []*api.Field{
					{
						Name:  "name",
						Typez: api.TypezString,
					},
					{
						Name:     "secret",
						Typez:    api.TypezMessage,
						TypezID:  ".google.cloud.secretmanager.v1.Secret",
						Optional: true,
					},
					{
						Name:  "project",
						Typez: api.TypezString,
					},
					{
						Name:     "location",
						Typez:    api.TypezString,
						Optional: true,
					},
				},
			}

			iam := &api.Service{
				Name: "SecretManagerService",
				Methods: []*api.Method{
					{
						Name:        "CreateSecret",
						InputTypeID: requestMessage.ID,
						InputType:   requestMessage,
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb:         "POST",
								PathTemplate: test.path,
							}},
						},
					},
				},
			}

			model := api.NewTestAPI([]*api.Message{requestMessage, secretMessage}, nil, []*api.Service{iam})
			model.PackageName = "google.cloud.secretmanager.v1"

			cfg := &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year": "2038",
				},
			}

			if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotBlock := extractBlock(t, contentStr, "let path = try { () throws -> String in", "    }()")
			if diff := cmp.Diff(test.wantBlock, gotBlock); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateService_Pagination(t *testing.T) {
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

	model := api.NewTestAPI([]*api.Message{inputType, outputType, secretType}, nil, []*api.Service{iam})
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

	verifyGeneratedService(t, outDir)
	verifyGeneratedRequest(t, outDir)
	verifyGeneratedResponse(t, outDir)
	verifyGeneratedMessage(t, outDir)
}

func verifyGeneratedService(t *testing.T, outDir string) {
	// Verify generated Service source code
	filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// TODO(https://github.com/googleapis/librarian/issues/5961): use extractBlock here
	wantMethodOverload := `public func listSecrets(byItem: ListSecretsRequest) throws -> some AsyncSequence<Secret, Error>
 {
      let listRpc = { (token: String) async throws -> ListSecretsResponse in
        var request = byItem
        request.pageToken = token
        return try await self.listSecrets(request: request)
      }
      return GoogleCloudGax.PaginatedResponseSequence(listRpc: listRpc)
    }`
	if !strings.Contains(contentStr, wantMethodOverload) {
		t.Fatalf("missing wanted method overload: \n%s\nfull content:\n%s", wantMethodOverload, contentStr)
	}
}

func verifyGeneratedRequest(t *testing.T, outDir string) {
	// Verify generated Request and Response Messages source code
	msgFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "ListSecretsRequest.swift")
	msgContent, err := os.ReadFile(msgFilename)
	if err != nil {
		t.Fatal(err)
	}
	msgContentStr := string(msgContent)

	gotRequestMessage := extractBlock(t, msgContentStr, "public struct ListSecretsRequest: ", "{")
	for _, p := range []string{"Codable", "Equatable", "GoogleCloudWkt._AnyPackable", "Sendable"} {
		if !strings.Contains(gotRequestMessage, p) {
			t.Errorf("expected %q in ListSecretsRequest declaration, got: %s", p, gotRequestMessage)
		}
	}

}

func verifyGeneratedResponse(t *testing.T, outDir string) {
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
	wantGetItems := `public func _getPaginatedItems() -> [Secret] {
    return self.secrets
  }`
	if diff := cmp.Diff(wantGetItems, gotGetItems); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if !strings.Contains(respContentStr, "import GoogleCloudGax") {
		t.Errorf("expected ListSecretsResponse.swift to import GoogleCloudGax, got:\n%s", respContentStr)
	}
}

func verifyGeneratedMessage(t *testing.T, outDir string) {
	secretFilename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "Secret.swift")
	secretContent, err := os.ReadFile(secretFilename)
	if err != nil {
		t.Fatal(err)
	}
	secretContentStr := string(secretContent)
	if strings.Contains(secretContentStr, "import GoogleCloudGax") {
		t.Errorf("expected Secret.swift to NOT import GoogleCloudGax, got:\n%s", secretContentStr)
	}
}
