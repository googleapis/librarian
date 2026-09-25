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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateStorage_MultiModel(t *testing.T) {
	outDir := t.TempDir()

	// Storage v2 Messages & Service (Data/Metadata subset)
	bucket := api.NewTestMessage("Bucket").WithPackage("google.storage.v2")
	createBucketRequest := api.NewTestMessage("CreateBucketRequest").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("bucket_id").WithType(api.TypezString),
		)
	listBucketsRequest := api.NewTestMessage("ListBucketsRequest").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("page_size").WithType(api.TypezInt32),
			api.NewTestField("page_token").WithType(api.TypezString),
		)

	listBucketsResponse := api.NewTestMessage("ListBucketsResponse").
		WithPackage("google.storage.v2").
		WithFields(
			api.NewTestField("buckets").
				WithMessageType(bucket).
				WithRepeated(),
			api.NewTestField("next_page_token").WithType(api.TypezString),
		)

	storageService := api.NewTestService("Storage").
		WithPackage("google.storage.v2").
		WithMethods(
			api.NewTestMethod("CreateBucket").
				WithInput(createBucketRequest).
				WithOutput(bucket).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("buckets")),
			api.NewTestMethod("ListBuckets").
				WithInput(listBucketsRequest).
				WithOutput(listBucketsResponse).
				WithVerb("GET").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("buckets")),
		).
		WithDefaultHost("storage.googleapis.com")

	storageModel := api.NewTestAPI([]*api.Message{bucket, createBucketRequest, listBucketsRequest, listBucketsResponse}, nil, []*api.Service{storageService})
	if err := api.CrossReference(storageModel); err != nil {
		t.Fatal(err)
	}
	api.UpdateMethodPagination(nil, storageModel)

	// StorageControl Messages & Service
	folder := api.NewTestMessage("Folder").WithPackage("google.storage.control.v2")
	createFolderRequest := api.NewTestMessage("CreateFolderRequest").
		WithPackage("google.storage.control.v2").
		WithFields(
			api.NewTestField("parent").WithType(api.TypezString),
			api.NewTestField("folder_id").WithType(api.TypezString),
		)
	policy := api.NewTestMessage("Policy").WithPackage("google.iam.v1")
	getIamPolicyRequest := api.NewTestMessage("GetIamPolicyRequest").
		WithPackage("google.iam.v1").
		WithFields(
			api.NewTestField("resource").WithType(api.TypezString),
		)
	renameFolderRequest := api.NewTestMessage("RenameFolderRequest").
		WithPackage("google.storage.control.v2").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)
	renameFolderMetadata := api.NewTestMessage("RenameFolderMetadata").
		WithPackage("google.storage.control.v2")
	operation := api.NewTestMessage("Operation").
		WithPackage("google.longrunning")
	getOperationRequest := api.NewTestMessage("GetOperationRequest").
		WithPackage("google.longrunning").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
		)

	controlService := api.NewTestService("StorageControl").
		WithPackage("google.storage.control.v2").
		WithMethods(
			api.NewTestMethod("CreateFolder").
				WithInput(createFolderRequest).
				WithOutput(folder).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("folders")),
			api.NewTestMethod("GetIamPolicy").
				WithInput(getIamPolicyRequest).
				WithOutput(policy).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("resource").WithLiteral(":getIamPolicy")),
			api.NewTestMethod("RenameFolder").
				WithInput(renameFolderRequest).
				WithOutput(operation).
				WithVerb("POST").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("name").WithLiteral(":rename")).
				WithOperationInfo(&api.OperationInfo{
					ResponseTypeID: folder.ID,
					MetadataTypeID: renameFolderMetadata.ID,
				}),
			api.NewTestMethod("GetOperation").
				WithInput(getOperationRequest).
				WithOutput(operation).
				WithVerb("GET").
				WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")),
		).
		WithDefaultHost("storage.googleapis.com")

	controlModel := api.NewTestAPI([]*api.Message{folder, createFolderRequest, renameFolderRequest, renameFolderMetadata}, nil, []*api.Service{controlService})
	controlModel.AddMessage(policy)
	controlModel.AddMessage(getIamPolicyRequest)
	controlModel.AddMessage(operation)
	controlModel.AddMessage(getOperationRequest)
	if err := api.CrossReference(controlModel); err != nil {
		t.Fatal(err)
	}

	// Module definitions
	storageModule := &config.SwiftModule{
		Output:     filepath.Join(outDir, "Storage"),
		ModuleType: "grpc-client",
		ModulePath: "StorageProtos",
	}
	controlModule := &config.SwiftModule{
		Output:     filepath.Join(outDir, "Control"),
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}

	swiftPkg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleGax", RequiredByServices: true},
		{Name: "GoogleAuth", RequiredByServices: true},
		{Name: "GoogleIAMV1", ApiPackage: "google.iam.v1"},
		{Name: "GoogleLongRunning", ApiPackage: "google.longrunning"},
	})
	swiftPkg.PackageNameOverride = "GoogleCloudStorage"
	swiftPkg.LibraryNameOverride = "GoogleCloudStorage"
	library := &config.Library{
		Name:          "google-cloud-storage",
		CopyrightYear: "2026",
		Swift:         swiftPkg,
	}

	if err := Generate(t.Context(), storageModel, storageModule.Output, library, storageModule); err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), controlModel, controlModule.Output, library, controlModule); err != nil {
		t.Fatal(err)
	}
	if err := GenerateStorage(t.Context(), filepath.Join(outDir, "Control"), storageModel, storageModule, controlModel, controlModule, library); err != nil {
		t.Fatal(err)
	}

	// 1. Verify StorageControlProtocol.swift in Control/
	protocolPath := filepath.Join(outDir, "Control", "StorageControlProtocol.swift")
	protocolContent, err := os.ReadFile(protocolPath)
	if err != nil {
		t.Fatalf("StorageControlProtocol.swift not generated: %v", err)
	}
	protocolStr := string(protocolContent)

	if !strings.Contains(protocolStr, "// Copyright 2026 Google LLC") {
		t.Errorf("StorageControlProtocol.swift missing Copyright header:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "public protocol StorageControlProtocol: Sendable {") {
		t.Errorf("StorageControlProtocol.swift missing StorageControlProtocol declaration:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "func createBucket(") ||
		!strings.Contains(protocolStr, "func listBuckets(") ||
		!strings.Contains(protocolStr, "func createFolder(") ||
		!strings.Contains(protocolStr, "func getIamPolicy(") {
		t.Errorf("StorageControlProtocol.swift missing unified methods:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "request: ListBucketsRequest, options: GoogleGax.RequestOptions") ||
		!strings.Contains(protocolStr, "any AsyncSequence<Bucket, Swift.Error> & Sendable") ||
		!strings.Contains(protocolStr, "return GoogleGax.PaginatedResponseSequence(listRpc: listRpc)") {
		t.Errorf("StorageControlProtocol.swift missing paginated helper method:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "public import GoogleIAMV1") {
		t.Errorf("StorageControlProtocol.swift missing public import GoogleIAMV1:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "public import GoogleGax") {
		t.Errorf("StorageControlProtocol.swift missing public import GoogleGax:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "func createBucket(request: CreateBucketRequest) async throws") {
		t.Errorf("StorageControlProtocol.swift missing convenience overload without options:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "func listBucketsByItems(\n  request: ListBucketsRequest\n)") ||
		!strings.Contains(protocolStr, "self.listBucketsByItems(request: request, options: .init())") {
		t.Errorf("StorageControlProtocol.swift missing paginated convenience overload without options:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "func renameFolderPollingUntilDone(request: RenameFolderRequest) async throws -> any GoogleGax.PollableOperation<Folder>") {
		t.Errorf("StorageControlProtocol.swift missing LRO method requirement:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "renameFolderPollingUntilDone(") ||
		!strings.Contains(protocolStr, "request: RenameFolderRequest, options: GoogleGax.RequestOptions") ||
		!strings.Contains(protocolStr, ") async throws -> any GoogleGax.PollableOperation<Folder>") {
		t.Errorf("StorageControlProtocol.swift missing LRO method with options requirement:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "extension StorageControlProtocol {") {
		t.Errorf("StorageControlProtocol.swift missing StorageControlProtocol extension:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "try await self.createBucket(request: request, options: .init())") {
		t.Errorf("StorageControlProtocol.swift missing default implementation forwarding to options:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "try await self.renameFolderPollingUntilDone(request: request, options: .init())") {
		t.Errorf("StorageControlProtocol.swift missing LRO default implementation forwarding to options:\n%s", protocolStr)
	}

	// 2. Verify StorageControlClient.swift in Control/
	clientPath := filepath.Join(outDir, "Control", "StorageControlClient.swift")
	clientContent, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("StorageControlClient.swift not generated: %v", err)
	}
	clientStr := string(clientContent)

	if !strings.Contains(clientStr, "// Copyright 2026 Google LLC") {
		t.Errorf("StorageControlClient.swift missing Copyright header:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "public final class StorageControlClient: StorageControlProtocol, Sendable {") {
		t.Errorf("StorageControlClient.swift missing class declaration:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "@_spi(GoogleCloudInternal) public import GoogleGax") {
		t.Errorf("StorageControlClient.swift missing @_spi(GoogleCloudInternal) public import GoogleGax:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "@_spi(GoogleCloudInternal) public import GoogleIAMV1") {
		t.Errorf("StorageControlClient.swift missing @_spi(GoogleCloudInternal) public import GoogleIAMV1:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "private let storage: any Clients.StorageStub") ||
		!strings.Contains(clientStr, "private let control: any Clients.StorageControlStub") {
		t.Errorf("StorageControlClient.swift missing private stub fields:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "let pollingErrorPolicy: any GoogleGax.PollingErrorPolicy") ||
		!strings.Contains(clientStr, "let pollingBackoffPolicy: any GoogleGax.BackoffPolicy") {
		t.Errorf("StorageControlClient.swift missing polling policy fields:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "let sharedGrpcClient = try GoogleGaxGRPC._GRPCClient(") ||
		!strings.Contains(clientStr, `withDefaultEndpoint: "https://storage.googleapis.com"`) {
		t.Errorf("StorageControlClient.swift missing shared _GRPCClient initialization:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "options.retryPolicy = StorageBaseRetryPolicy.defaultPolicy") {
		t.Errorf("StorageControlClient.swift missing StorageBaseRetryPolicy default initialization:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "self.pollingErrorPolicy = options.pollingErrorPolicy") ||
		!strings.Contains(clientStr, "self.pollingBackoffPolicy = options.pollingBackoffPolicy") {
		t.Errorf("StorageControlClient.swift missing polling policy initialization:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "var storageStub: any Clients.StorageStub = Clients.StorageTransport(sharedGrpcClient)") ||
		!strings.Contains(clientStr, "var controlStub: any Clients.StorageControlStub = Clients.StorageControlTransport(sharedGrpcClient)") {
		t.Errorf("StorageControlClient.swift missing transport initialization with shared gRPC client:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "storageStub = Clients.StorageRetry(storageStub, options: options)") ||
		!strings.Contains(clientStr, "controlStub = Clients.StorageControlRetry(controlStub, options: options)") {
		t.Errorf("StorageControlClient.swift missing retry decorators:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "try await self.storage.createBucket(request: request, options: options)") ||
		!strings.Contains(clientStr, "try await self.control.createFolder(request: request, options: options)") ||
		!strings.Contains(clientStr, "try await self.control.getIamPolicy(request: request, options: options)") {
		t.Errorf("StorageControlClient.swift missing method delegation:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "public func renameFolderPollingUntilDone(") ||
		!strings.Contains(clientStr, "request: RenameFolderRequest, options: GoogleGax.RequestOptions") {
		t.Errorf("StorageControlClient.swift missing LRO helper method:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "let rawOp = try await self.renameFolder(request: request, options: options)") ||
		!strings.Contains(clientStr, "let op = try await self.getOperation(request: .init().with { $0.name = rawOp.name }, options: options)") ||
		!strings.Contains(clientStr, "return GoogleGax._PollableOperationImpl(") {
		t.Errorf("StorageControlClient.swift missing LRO helper implementation:\n%s", clientStr)
	}

	// 3. Verify Storage+Stub.swift and Storage+Transport.swift generated in Storage/
	storageStubPath := filepath.Join(outDir, "Storage", "Storage+Stub.swift")
	storageStubContent, err := os.ReadFile(storageStubPath)
	if err != nil {
		t.Fatalf("Storage+Stub.swift not generated: %v", err)
	}
	storageStubStr := string(storageStubContent)
	if !strings.Contains(storageStubStr, "protocol StorageStub") {
		t.Errorf("Storage+Stub.swift missing StorageStub:\n%s", storageStubStr)
	}

	storageTransportPath := filepath.Join(outDir, "Storage", "Storage+Transport.swift")
	storageTransportContent, err := os.ReadFile(storageTransportPath)
	if err != nil {
		t.Fatalf("Storage+Transport.swift not generated: %v", err)
	}
	storageTransportStr := string(storageTransportContent)
	if !strings.Contains(storageTransportStr, "class StorageTransport: StorageStub") ||
		!strings.Contains(storageTransportStr, `path: "/google.storage.v2.Storage/CreateBucket"`) {
		t.Errorf("Storage+Transport.swift missing gRPC transport details:\n%s", storageTransportStr)
	}

	// 4. Verify StorageControl+Stub.swift and StorageControl+Transport.swift generated in Control/
	controlStubPath := filepath.Join(outDir, "Control", "StorageControl+Stub.swift")
	controlStubContent, err := os.ReadFile(controlStubPath)
	if err != nil {
		t.Fatalf("StorageControl+Stub.swift not generated: %v", err)
	}
	controlStubStr := string(controlStubContent)
	if !strings.Contains(controlStubStr, "protocol StorageControlStub") {
		t.Errorf("StorageControl+Stub.swift missing StorageControlStub:\n%s", controlStubStr)
	}
}
