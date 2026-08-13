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

func TestGenerateStub_Structure(t *testing.T) {
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
	service := &api.Service{
		Name:    "Protocol",
		ID:      ".test.Prototocol",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "GetThing",
				ID:           ".test.IAM.CreateRole",
				InputTypeID:  ".test.Request",
				InputType:    request,
				OutputTypeID: ".test.Response",
				OutputType:   response,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:       "SomeTestPackage",
			ApiPackage: "test",
		},
	})
	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	stubFilename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Protocol+Stub.swift")
	stubContent, err := os.ReadFile(stubFilename)
	if err != nil {
		t.Fatal(err)
	}
	stubContentStr := string(stubContent)

	got := extractBlock(t, stubContentStr, `  protocol ProtocolStub {`, "\n"+`  }`)
	want := `  protocol ProtocolStub {
    func getThing(
    request: SomeTestPackage.Request, options: GoogleCloudGax.RequestOptions
) async throws -> SomeTestPackage.Response

  }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	transportFilename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Protocol+Transport.swift")
	transportContent, err := os.ReadFile(transportFilename)
	if err != nil {
		t.Fatal(err)
	}
	transportContentStr := string(transportContent)

	got = extractBlock(t, transportContentStr, `  class ProtocolTransport: `, `HTTPClient`)
	want = `  class ProtocolTransport: ProtocolStub {
    let inner: GoogleCloudGax._HTTPClient`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, transportContentStr, `return try await req.rpc(`, ".get()\n    }")
	want = `return try await req.rpc(
        SomeTestPackage.Response.self, timeout: options.attemptTimeout
      ).get()
    }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, transportContentStr, `URLQueryItem(name: "$alt",`, ")")
	want = `URLQueryItem(name: "$alt", value: "json;enum-encoding=int")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateStub_QueryParameters(t *testing.T) {
	outDir := t.TempDir()

	oneof := &api.OneOf{Name: "expiration"}
	oneofField := &api.Field{
		Name:     "ttl_days",
		JSONName: "ttlDays",
		ID:       ".google.test.Request.ttl_days",
		Typez:    api.TypezString,
		IsOneOf:  true,
		Group:    oneof,
	}
	oneof.Fields = []*api.Field{oneofField}

	request := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
		Fields: []*api.Field{
			oneofField,
			{
				Name:     "project",
				JSONName: "project",
				ID:       ".google.test.Request.project",
				Typez:    api.TypezString,
			},
			{
				Name:     "enable",
				JSONName: "enable",
				ID:       ".google.test.Request.enable",
				Typez:    api.TypezBool,
			},
		},
		OneOfs: []*api.OneOf{oneof},
	}
	response := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".test.Service",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "GetThing",
				ID:           ".test.Service.GetThing",
				InputTypeID:  ".test.Request",
				InputType:    request,
				OutputTypeID: ".test.Response",
				OutputType:   response,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project"),
						QueryParameters: map[string]bool{
							"ttl_days": true,
							"enable":   true,
						},
					}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})
	model.PackageName = "test"

	swiftCfg := swiftConfig(t, []config.SwiftDependency{})
	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "Test", "Service+Transport.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	got := extractBlock(t, contentStr, `contentsOf: try encoder.encode(request.enable`, `)`)
	want := `contentsOf: try encoder.encode(request.enable, prefix: "enable")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, contentStr, `request.expiration.flatMap {`, `prefix: "ttlDays")`)
	want = `request.expiration.flatMap { (oneof) -> Swift.String? in
            if case let .ttlDays(v) = oneof { v } else { nil }
          }, prefix: "ttlDays")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateStub_Discovery(t *testing.T) {
	testdataDir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: config.SpecDiscovery,
		ServiceConfig:       filepath.Join(testdataDir, "googleapis/google/cloud/compute/v1/small-compute_v1.yaml"),
		SpecificationSource: filepath.Join(testdataDir, "discovery/small-compute.v1.json"),
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		SpecificationFormat: config.SpecDiscovery,
		Swift:               swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(outDir, "Sources", "GoogleCloudComputeV1", "Addresses+Transport.swift"))
	if err != nil {
		t.Fatal(err)
	}
	got := extractBlock(t, string(contentBytes), `URLQueryItem(name: "$alt",`, ")")
	want := `URLQueryItem(name: "$alt", value: "json")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateStub_Grpc(t *testing.T) {
	outDir := t.TempDir()

	parentField := &api.Field{
		Name:     "parent",
		JSONName: "parent",
		ID:       ".google.storage.control.v2.CreateFolderRequest.parent",
		Typez:    api.TypezString,
	}
	folderField := &api.Field{
		Name:     "folder",
		JSONName: "folder",
		ID:       ".google.storage.control.v2.CreateFolderRequest.folder",
		Typez:    api.TypezMessage,
		TypezID:  ".google.storage.control.v2.Folder",
		Optional: true,
	}
	nameField := &api.Field{
		Name:     "name",
		JSONName: "name",
		ID:       ".google.storage.control.v2.DeleteFolderRequest.name",
		Typez:    api.TypezString,
	}

	createFolderRequest := &api.Message{
		Name:    "CreateFolderRequest",
		ID:      ".google.storage.control.v2.CreateFolderRequest",
		Package: "google.storage.control.v2",
		Fields:  []*api.Field{parentField, folderField},
	}
	deleteFolderRequest := &api.Message{
		Name:    "DeleteFolderRequest",
		ID:      ".google.storage.control.v2.DeleteFolderRequest",
		Package: "google.storage.control.v2",
		Fields:  []*api.Field{nameField},
	}
	folder := &api.Message{
		Name:    "Folder",
		ID:      ".google.storage.control.v2.Folder",
		Package: "google.storage.control.v2",
	}
	empty := &api.Message{
		Name:    "Empty",
		ID:      ".google.protobuf.Empty",
		Package: "google.protobuf",
	}
	operation := &api.Message{
		Name:    "Operation",
		ID:      ".google.longrunning.Operation",
		Package: "google.longrunning",
	}
	getOperationRequest := &api.Message{
		Name:    "GetOperationRequest",
		ID:      ".google.longrunning.GetOperationRequest",
		Package: "google.longrunning",
	}

	service := &api.Service{
		Name:        "StorageControl",
		ID:          ".google.storage.control.v2.StorageControl",
		Package:     "google.storage.control.v2",
		DefaultHost: "storage.googleapis.com",
		Methods: []*api.Method{
			{
				Name:         "CreateFolder",
				ID:           ".google.storage.control.v2.StorageControl.CreateFolder",
				InputTypeID:  createFolderRequest.ID,
				InputType:    createFolderRequest,
				OutputTypeID: folder.ID,
				OutputType:   folder,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "POST",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithLiteral("projects").WithVariableNamed("parent").WithLiteral("folders"),
					}},
				},
			},
			{
				Name:         "DeleteFolder",
				ID:           ".google.storage.control.v2.StorageControl.DeleteFolder",
				InputTypeID:  deleteFolderRequest.ID,
				InputType:    deleteFolderRequest,
				OutputTypeID: empty.ID,
				OutputType:   empty,
				ReturnsEmpty: true,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "DELETE",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithLiteral("projects").WithVariableNamed("name"),
					}},
				},
			},
			{
				Name:            "GetOperation",
				ID:              ".google.longrunning.Operations.GetOperation",
				SourceServiceID: ".google.longrunning.Operations",
				SourceService: &api.Service{
					Name:    "Operations",
					Package: "google.longrunning",
					ID:      ".google.longrunning.Operations",
				},
				InputTypeID:  getOperationRequest.ID,
				InputType:    getOperationRequest,
				OutputTypeID: operation.ID,
				OutputType:   operation,
			},
		},
	}

	operationsService := &api.Service{
		Name:    "Operations",
		ID:      ".google.longrunning.Operations",
		Package: "google.longrunning",
	}
	model := api.NewTestAPI([]*api.Message{createFolderRequest, deleteFolderRequest, folder, empty, operation, getOperationRequest}, nil, []*api.Service{service, operationsService})
	model.PackageName = "google.storage.control.v2"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	module := &config.SwiftModule{
		Output:     outDir,
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	swiftPkg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleLongRunning", ApiPackage: "google.longrunning"},
	})
	swiftPkg.PackageNameOverride = "GoogleCloudStorage"
	swiftPkg.LibraryNameOverride = "GoogleCloudStorage"
	library := &config.Library{
		Name:  "google-cloud-storage",
		Swift: swiftPkg,
	}

	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	stubFilename := filepath.Join(outDir, "StorageControl+Stub.swift")
	stubContent, err := os.ReadFile(stubFilename)
	if err != nil {
		t.Fatal(err)
	}
	stubStr := string(stubContent)
	if !strings.Contains(stubStr, "protocol StorageControlStub {") {
		t.Errorf("stub file missing StorageControlStub protocol:\n%s", stubStr)
	}
	if !strings.Contains(stubStr, "func createFolder(") || !strings.Contains(stubStr, "func deleteFolder(") || !strings.Contains(stubStr, "func getOperation(") {
		t.Errorf("stub file missing methods:\n%s", stubStr)
	}

	transportFilename := filepath.Join(outDir, "StorageControl+Transport.swift")
	transportContent, err := os.ReadFile(transportFilename)
	if err != nil {
		t.Fatal(err)
	}
	transportStr := string(transportContent)

	// Check imports
	if !strings.Contains(transportStr, "@_spi(GoogleCloudInternal) import GoogleCloudGax") {
		t.Errorf("transport missing @_spi(GoogleCloudInternal) import GoogleCloudGax:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "internal import StorageControlProtos") {
		t.Errorf("transport missing internal import StorageControlProtos:\n%s", transportStr)
	}

	// Check gRPC Transport class definition and inner client
	if !strings.Contains(transportStr, "class StorageControlTransport: StorageControlStub {") {
		t.Errorf("transport missing StorageControlTransport class declaration:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "let inner: GoogleCloudGaxGRPC._GRPCClient") {
		t.Errorf("transport missing inner: GoogleCloudGaxGRPC._GRPCClient field:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `self.inner = try GoogleCloudGaxGRPC._GRPCClient(`) ||
		!strings.Contains(transportStr, `withDefaultEndpoint: "https://storage.googleapis.com"`) {
		t.Errorf("transport missing _GRPCClient initialization with default endpoint:\n%s", transportStr)
	}

	// Check method body conversions and generic inner.execute dispatch
	if !strings.Contains(transportStr, "let protoRequest = try request.toProto()") {
		t.Errorf("transport missing request.toProto() call:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "pathVariable0.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed)") {
		t.Errorf("transport missing addingPercentEncoding for routing params:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `path: "/google.storage.control.v2.StorageControl/CreateFolder"`) {
		t.Errorf("transport missing CreateFolder gRPC path in execute:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "return try Folder(proto: protoResponse)") {
		t.Errorf("transport missing Folder(proto: protoResponse) return:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "let _: SwiftProtobuf.Google_Protobuf_Empty = try await self.inner.execute(") ||
		!strings.Contains(transportStr, `path: "/google.storage.control.v2.StorageControl/DeleteFolder"`) {
		t.Errorf("transport missing empty response deleteFolder call:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `path: "/google.longrunning.Operations/GetOperation"`) {
		t.Errorf("transport missing operations GetOperation gRPC path in execute:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `routingParams.append("parent=\(encoded)")`) {
		t.Errorf("transport missing routing parameter extraction in method body:\n%s", transportStr)
	}
}
