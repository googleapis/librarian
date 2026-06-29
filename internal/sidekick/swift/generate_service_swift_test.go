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
	"bytes"
	"fmt"
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

	// We need explicit Package and ID fields because we generate both messages
	// and services.
	iam := &api.Service{Name: "IAM", Package: "test", ID: ".test.IAM"}
	secretManager := &api.Service{Name: "SecretManagerService", Package: "test", ID: ".test.SecretManagerService"}
	clash0 := &api.Message{Name: "InstanceSettings", Package: "test", ID: ".test.InstanceSettings"}
	clash1 := &api.Service{Name: "instanceSettings", Package: "test", ID: ".test.instanceSettings"}

	model := api.NewTestAPI([]*api.Message{clash0}, nil, []*api.Service{iam, secretManager, clash1})
	model.PackageName = "test"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleTest")
	wantFiles := []string{
		"IAM.swift",
		"Clients.swift",
		"SecretManagerService.swift",
		"SecretManagerService+Stub.swift",
		"SecretManagerService+Logging.swift",
		"SecretManagerService+Retry.swift",
		"InstanceSettings.swift",
		"instanceSettings+000.swift",
		"instanceSettings+Stub.swift",
		"instanceSettings+Logging.swift",
		"instanceSettings+Retry.swift",
	}
	for _, expected := range wantFiles {
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

	gotBlock := extractBlock(t, contentStr, "public protocol ", "{")
	wantBlock := `public protocol ProtocolProtocol {`
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_Delegation(t *testing.T) {
	outDir := t.TempDir()

	request := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
	}
	response := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	iam := &api.Service{
		Name:    "IAM",
		ID:      ".test.IAM",
		Package: "test",

		Methods: []*api.Method{
			{
				Name:         "CreateRole",
				ID:           ".test.IAM.CreateRole",
				InputTypeID:  ".test.Request",
				InputType:    request,
				OutputTypeID: ".test.Response",
				OutputType:   response,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v1"),
						},
					},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:       "SomeTestPackage",
			ApiPackage: "test",
		},
	})
	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "IAM.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	for _, want := range []string{
		"let inner: any Clients.IAMStub",
		"var inner: any Clients.IAMStub = try Clients.IAMTransport(options)",
		"try await self.inner.createRole(request: request, options: options)",
	} {
		if !strings.Contains(contentStr, want) {
			t.Errorf("expected %q in IAM.swift, got:\n%s", want, contentStr)
		}
	}
}

func TestGenerateService_SnippetFiles(t *testing.T) {
	outDir := t.TempDir()

	packageName := "google.cloud.test.v1"
	dummyMessage := &api.Message{Name: "DummyMessage", Package: packageName}
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
	model.PackageName = packageName

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
				Name:       "TestMethod",
				InputType:  inputMessage,
				OutputType: externalMessage,
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
import GoogleCloudExternalV1
import GoogleCloudGax`

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
			wantBlock: `let path = try { () throws -> Swift.String in
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
			wantBlock: `let path = try { () throws -> Swift.String in
        guard let pathVariable0 = request.name as Swift.String?, !pathVariable0.isEmpty else {
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
			wantBlock: `let path = try { () throws -> Swift.String in
        guard let pathVariable0 = request.project as Swift.String?, !pathVariable0.isEmpty else {
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

			filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService+Stub.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotBlock := extractBlock(t, contentStr, "let path = try { () throws -> Swift.String in", "    }()")
			if diff := cmp.Diff(test.wantBlock, gotBlock); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateService_Pagination(t *testing.T) {
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

			itemField := &api.Field{Name: "secrets", JSONName: "secrets", Typez: api.TypezMessage, TypezID: ".google.cloud.secretmanager.v1.Secret", Repeated: true}
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
			verifyGeneratedResponse(t, outDir, test.wantNextPageToken)
			verifyGeneratedMessage(t, outDir)
		})
	}
}

func verifyGeneratedService(t *testing.T, outDir string) {
	t.Helper()
	// Verify generated Service source code
	filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotMethodOverload := extractBlock(t, contentStr, `  public func listSecrets(
    byItem: ListSecretsRequest, options: `, "\n  }")
	wantMethodOverload := `  public func listSecrets(
    byItem: ListSecretsRequest, options: GoogleCloudGax.RequestOptions
) throws -> any AsyncSequence<Secret, Swift.Error>
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

func verifyGeneratedRequest(t *testing.T, outDir string) {
	t.Helper()
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

func verifyGeneratedResponse(t *testing.T, outDir string, wantNextPageToken string) {
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
	wantGetItems := `public func _getPaginatedItems() -> [Secret] {
    return self.secrets
  }`
	if diff := cmp.Diff(wantGetItems, gotGetItems); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	gotNextPageToken := extractBlock(t, respContentStr, "public func _nextPageToken()", "  }")
	if diff := cmp.Diff(wantNextPageToken, gotNextPageToken); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	if !strings.Contains(respContentStr, "import GoogleCloudGax") {
		t.Errorf("expected ListSecretsResponse.swift to import GoogleCloudGax, got:\n%s", respContentStr)
	}
}

func verifyGeneratedMessage(t *testing.T, outDir string) {
	t.Helper()
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

func TestGenerateService_LRO(t *testing.T) {
	outDir := t.TempDir()

	operationType := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
	}

	workflowType := &api.Message{
		Name:    "Workflow",
		Package: "google.cloud.workflows.v1",
		ID:      ".google.cloud.workflows.v1.Workflow",
	}

	metadataType := &api.Message{
		Name:    "OperationMetadata",
		Package: "google.cloud.workflows.v1",
		ID:      ".google.cloud.workflows.v1.OperationMetadata",
	}

	inputType := &api.Message{
		Name:    "CreateWorkflowRequest",
		Package: "google.cloud.workflows.v1",
		ID:      ".google.cloud.workflows.v1.CreateWorkflowRequest",
	}

	getOperationInputType := &api.Message{
		Name:    "GetOperationRequest",
		Package: "google.longrunning",
		ID:      ".google.longrunning.GetOperationRequest",
	}

	workflows := &api.Service{
		Name: "WorkflowsService",
		Methods: []*api.Method{
			{
				Name:          "CreateWorkflow",
				Documentation: "Creates a workflow.",
				InputTypeID:   inputType.ID,
				InputType:     inputType,
				OutputTypeID:  operationType.ID,
				OutputType:    operationType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "POST",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows"),
					}},
				},
				IsLRO: true,
				OperationInfo: &api.OperationInfo{
					ResponseTypeID: workflowType.ID,
					MetadataTypeID: metadataType.ID,
				},
			},
			{
				Name:         "GetOperation",
				InputTypeID:  getOperationInputType.ID,
				InputType:    getOperationInputType,
				OutputTypeID: operationType.ID,
				OutputType:   operationType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations"),
					}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputType, workflowType, metadataType, operationType, getOperationInputType}, nil, []*api.Service{workflows})
	model.PackageName = "google.cloud.workflows.v1"

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
			ApiPackage: "google.longrunning",
			Name:       "GoogleCloudLongrunningV1",
		},
		{
			ApiPackage: "google.rpc",
			Name:       "GoogleRpc",
		},
	})

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudWorkflowsV1", "WorkflowsService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	wantContains := []string{
		"import GoogleRpc",
		"public func createWorkflow(withPolling: CreateWorkflowRequest) async throws -> any GoogleCloudGax.PollableOperation<Workflow>",
		"GoogleCloudGax._PollableOperationImpl(initialState: initialState, poll: poll)",
		"self.getOperation(request: .init().with { $0.name = rawOp.name }, options: options)",
	}
	for _, want := range wantContains {
		if !strings.Contains(contentStr, want) {
			t.Errorf("expected %q in WorkflowsService.swift, got:\n%s", want, contentStr)
		}
	}
}

func TestGenerateService_LRO_Empty(t *testing.T) {
	outDir := t.TempDir()

	operationType := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
	}

	metadataType := &api.Message{
		Name:    "OperationMetadata",
		Package: "google.cloud.workflows.v1",
		ID:      ".google.cloud.workflows.v1.OperationMetadata",
	}

	inputType := &api.Message{
		Name:    "DeleteWorkflowRequest",
		Package: "google.cloud.workflows.v1",
		ID:      ".google.cloud.workflows.v1.DeleteWorkflowRequest",
	}

	getOperationInputType := &api.Message{
		Name:    "GetOperationRequest",
		Package: "google.longrunning",
		ID:      ".google.longrunning.GetOperationRequest",
	}

	workflows := &api.Service{
		Name: "WorkflowsService",
		Methods: []*api.Method{
			{
				Name:          "DeleteWorkflow",
				Documentation: "Deletes a workflow.",
				InputTypeID:   inputType.ID,
				InputType:     inputType,
				OutputTypeID:  operationType.ID,
				OutputType:    operationType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "DELETE",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("workflows"),
					}},
				},
				IsLRO: true,
				OperationInfo: &api.OperationInfo{
					ResponseTypeID: ".google.protobuf.Empty",
					MetadataTypeID: metadataType.ID,
				},
			},
			{
				Name:         "GetOperation",
				InputTypeID:  getOperationInputType.ID,
				InputType:    getOperationInputType,
				OutputTypeID: operationType.ID,
				OutputType:   operationType,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations"),
					}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputType, metadataType, operationType, getOperationInputType}, nil, []*api.Service{workflows})
	model.PackageName = "google.cloud.workflows.v1"

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
			ApiPackage: "google.longrunning",
			Name:       "GoogleCloudLongrunningV1",
		},
		{
			ApiPackage: "google.rpc",
			Name:       "GoogleRpc",
		},
	})

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudWorkflowsV1", "WorkflowsService.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	wantContains := []string{
		"public func deleteWorkflow(withPolling: DeleteWorkflowRequest) async throws -> any GoogleCloudGax.PollableOperation<Void>",
		"GoogleCloudGax._PollableOperationImpl<Void>",
	}
	for _, want := range wantContains {
		if !strings.Contains(contentStr, want) {
			t.Errorf("expected %q in WorkflowsService.swift, got:\n%s", want, contentStr)
		}
	}
}

func TestGenerateDiscoveryService_Files(t *testing.T) {
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecDiscovery,
		ServiceConfig:       filepath.Join(testdataDir, "googleapis/google/cloud/compute/v1/small-compute_v1.yaml"),
		SpecificationSource: filepath.Join(testdataDir, "discovery/small-compute.v1.json"),
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	// Verify files
	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudComputeV1")

	for _, test := range []struct {
		filename    string
		serviceName string
		structName  string
	}{
		{
			filename:    "AcceleratorTypes+Requests.swift",
			serviceName: "AcceleratorTypes",
			structName:  "ListRequest",
		},
		{
			filename:    "Addresses+Requests.swift",
			serviceName: "Addresses",
			structName:  "DeleteRequest",
		},
		{
			filename:    "instances+Requests.swift",
			serviceName: "Instances",
			structName:  "GetRequest",
		},
	} {
		t.Run(test.serviceName, func(t *testing.T) {
			filename := filepath.Join(expectedDir, test.filename)
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			// Verify it contains an extension to the right ${ServiceName}Client type.
			wantExtension := fmt.Appendf(nil, "extension %sClient {", test.serviceName)
			if !bytes.Contains(content, wantExtension) {
				t.Errorf("expected extension %q in %s, got:\n%s", wantExtension, filename, content)
			}

			// Verify the request struct definition appears in that file.
			wantStruct := fmt.Appendf(nil, "public struct %s: ", test.structName)
			if !bytes.Contains(content, wantStruct) {
				t.Errorf("expected struct %q in %s, got:\n%s", wantStruct, filename, content)
			}
		})
	}
}
