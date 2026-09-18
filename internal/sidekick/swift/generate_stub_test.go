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

	request := api.NewTestMessage("Request").WithPackage("test")
	response := api.NewTestMessage("Response").WithPackage("test")
	service := api.NewTestService("Protocol").
		WithPackage("google.cloud.test.v1").
		WithMethods(
			api.NewTestMethod("GetThing").
				WithInput(request).
				WithOutput(response).
				WithVerb("GET").
				WithPathTemplate(&api.PathTemplate{}),
		)

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.AddMessage(request)
	model.AddMessage(response)

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

	got := extractBlock(t, stubContentStr, `  protocol ProtocolStub: Sendable {`, "\n"+`  }`)
	want := `  protocol ProtocolStub: Sendable {
    func getThing(
    request: SomeTestPackage.Request, options: GoogleGax.RequestOptions
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

	got = extractBlock(t, transportContentStr, `  final class ProtocolTransport: `, `HTTPClient`)
	want = `  final class ProtocolTransport: ProtocolStub {
    let inner: GoogleGax._HTTPClient`
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

	oneof := api.NewTestOneOf("expiration").
		WithFields(
			api.NewTestField("ttl_days").WithType(api.TypezString),
		)

	request := api.NewTestMessage("Request").
		WithPackage("test").
		WithOneOfs(oneof).
		WithFields(
			api.NewTestField("project").WithType(api.TypezString),
			api.NewTestField("enable").WithType(api.TypezBool),
		)
	response := api.NewTestMessage("Response").WithPackage("test")

	getThing := api.NewTestMethod("GetThing").
		WithInput(request).
		WithOutput(response).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project")).
		WithQueryParameters(map[string]bool{
			"ttl_days": true,
			"enable":   true,
		})

	service := api.NewTestService("Service").
		WithPackage("test").
		WithMethods(getThing)

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})

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

	got = extractBlock(t, contentStr, `var query = [`, `query.append(`)
	want = `var query = [
            URLQueryItem(name: "$alt", value: "json;enum-encoding=int"),
          ]
          let encoder = GoogleGax._QueryParameterEncoder()
          query.append(`
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
		SpecificationFormat: config.SpecDiscovery,
		Swift:               swiftCfg,
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

	folder := api.NewTestMessage("Folder").
		WithPackage("google.storage.control.v2")
	createFolderRequest := api.NewTestMessage("CreateFolderRequest").
		WithPackage("google.storage.control.v2").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("folder").
				WithMessageType(folder).
				WithOptional(),
		)
	deleteFolderRequest := api.NewTestMessage("DeleteFolderRequest").
		WithPackage("google.storage.control.v2").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	empty := api.NewTestMessage("Empty").
		WithPackage("google.protobuf")
	operation := api.NewTestMessage("Operation").
		WithPackage("google.longrunning")
	getOperationRequest := api.NewTestMessage("GetOperationRequest").
		WithPackage("google.longrunning").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)

	deleteFolder := api.NewTestMethod("DeleteFolder").
		WithInput(deleteFolderRequest).
		WithOutput(empty)
	deleteFolder.PathInfo = nil
	deleteFolder.ReturnsEmpty = true
	deleteFolder.Routing = []*api.RoutingInfo{
		{
			Name: "bucket",
			Variants: []*api.RoutingInfoVariant{
				{
					FieldPath: []string{"name"},
				},
			},
		},
	}

	operationsService := api.NewTestService("Operations").
		WithPackage("google.longrunning")

	getOperation := api.NewTestMethod("GetOperation").
		WithInput(getOperationRequest).
		WithOutput(operation)
	getOperation.PathInfo = nil
	getOperation.SourceService = operationsService
	getOperation.SourceServiceID = operationsService.ID
	getOperation.ID = ".google.longrunning.Operations.GetOperation"

	service := api.NewTestService("StorageControl").
		WithPackage("google.storage.control.v2").
		WithMethods(
			api.NewTestMethod("CreateFolder").
				WithInput(createFolderRequest).
				WithOutput(folder).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithLiteral("projects").WithVariableNamed("parent").WithLiteral("folders")),
			deleteFolder,
			getOperation,
		)
	service.DefaultHost = "storage.googleapis.com"

	model := api.NewTestAPI([]*api.Message{folder, createFolderRequest, deleteFolderRequest}, nil, []*api.Service{service})
	model.AddService(operationsService)
	model.AddMessage(empty)
	model.AddMessage(operation)
	model.AddMessage(getOperationRequest)
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
	if !strings.Contains(stubStr, "protocol StorageControlStub: Sendable {") {
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
	if !strings.Contains(transportStr, "@_spi(GoogleCloudInternal) import GoogleGax") {
		t.Errorf("transport missing @_spi(GoogleCloudInternal) import GoogleGax:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "internal import StorageControlProtos") {
		t.Errorf("transport missing internal import StorageControlProtos:\n%s", transportStr)
	}

	// Check gRPC Transport class definition and inner client
	if !strings.Contains(transportStr, "final class StorageControlTransport: StorageControlStub {") {
		t.Errorf("transport missing StorageControlTransport class declaration:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "let inner: GoogleGaxGRPC._GRPCClient") {
		t.Errorf("transport missing inner: GoogleGaxGRPC._GRPCClient field:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `self.inner = try GoogleGaxGRPC._GRPCClient(`) ||
		!strings.Contains(transportStr, `withDefaultEndpoint: "https://storage.googleapis.com"`) {
		t.Errorf("transport missing _GRPCClient initialization with default endpoint:\n%s", transportStr)
	}

	// Check method body conversions and generic inner.execute dispatch
	if !strings.Contains(transportStr, "let protoRequest = try request.toProto()") {
		t.Errorf("transport missing request.toProto() call:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "GoogleGaxGRPC._RoutingMatcher.value(") {
		t.Errorf("transport missing GoogleGaxGRPC._RoutingMatcher.value for routing params:\n%s", transportStr)
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
	if !strings.Contains(transportStr, `("parent",`) {
		t.Errorf("transport missing parent routing parameter extraction in method body:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, `("bucket",`) {
		t.Errorf("transport missing bucket routing parameter extraction in method body:\n%s", transportStr)
	}
}

func TestGenerateStub_MultipleBindings(t *testing.T) {
	outDir := t.TempDir()

	request := api.NewTestMessage("Request").
		WithPackage("test").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("parent").WithType(api.TypezString),
		)
	response := api.NewTestMessage("Response").
		WithPackage("test")

	getResource := api.NewTestMethod("GetResource").
		WithInput(request).
		WithOutput(response).
		WithBindings(
			api.NewTestPathBinding("GET", (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("name")),
			api.NewTestPathBinding("POST", (&api.PathTemplate{}).
				WithLiteral("v1").
				WithVariableNamed("parent").
				WithLiteral("resources")),
		)

	service := api.NewTestService("Service").
		WithPackage("google.cloud.test.v1").
		WithMethods(getResource)

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.AddMessage(request)
	model.AddMessage(response)

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

	transportFilename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Service+Transport.swift")
	transportContent, err := os.ReadFile(transportFilename)
	if err != nil {
		t.Fatal(err)
	}
	transportStr := string(transportContent)

	if !strings.Contains(transportStr, "GoogleGax._RoutingMatcher.pathValue(") {
		t.Errorf("transport missing _RoutingMatcher.pathValue:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "GoogleGax._PathMismatchBuilder()") {
		t.Errorf("transport missing _PathMismatchBuilder:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "builder.maybeAdd(") {
		t.Errorf("transport missing builder.maybeAdd:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "throw GoogleGax.RequestError.binding(GoogleGax.BindingError(paths: paths))") {
		t.Errorf("transport missing throw RequestError.binding:\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "$0.setMethod(.GET)") {
		t.Errorf("transport missing $0.setMethod(.GET):\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "$0.setMethod(.POST)") {
		t.Errorf("transport missing $0.setMethod(.POST):\n%s", transportStr)
	}
	if !strings.Contains(transportStr, "configure(&req)") {
		t.Errorf("transport missing configure(&req):\n%s", transportStr)
	}
}
