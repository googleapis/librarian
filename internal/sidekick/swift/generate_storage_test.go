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
	bucket := &api.Message{
		Name:    "Bucket",
		ID:      ".google.storage.v2.Bucket",
		Package: "google.storage.v2",
	}
	createBucketRequest := &api.Message{
		Name:    "CreateBucketRequest",
		ID:      ".google.storage.v2.CreateBucketRequest",
		Package: "google.storage.v2",
		Fields: []*api.Field{
			{Name: "parent", JSONName: "parent", Typez: api.TypezString},
			{Name: "bucket_id", JSONName: "bucketId", Typez: api.TypezString},
		},
	}
	pageSizeField := &api.Field{Name: "page_size", JSONName: "pageSize", Typez: api.TypezInt32}
	pageTokenField := &api.Field{Name: "page_token", JSONName: "pageToken", Typez: api.TypezString}
	parentField := &api.Field{Name: "parent", JSONName: "parent", Typez: api.TypezString}
	listBucketsRequest := &api.Message{
		Name:    "ListBucketsRequest",
		ID:      ".google.storage.v2.ListBucketsRequest",
		Package: "google.storage.v2",
		Fields:  []*api.Field{parentField, pageSizeField, pageTokenField},
	}
	parentField.Parent = listBucketsRequest
	pageSizeField.Parent = listBucketsRequest
	pageTokenField.Parent = listBucketsRequest

	bucketsField := &api.Field{
		Name:        "buckets",
		JSONName:    "buckets",
		Typez:       api.TypezMessage,
		TypezID:     bucket.ID,
		MessageType: bucket,
		Repeated:    true,
	}
	nextPageTokenField := &api.Field{
		Name:     "next_page_token",
		JSONName: "nextPageToken",
		Typez:    api.TypezString,
	}
	listBucketsResponse := &api.Message{
		Name:    "ListBucketsResponse",
		ID:      ".google.storage.v2.ListBucketsResponse",
		Package: "google.storage.v2",
		Fields:  []*api.Field{bucketsField, nextPageTokenField},
	}
	bucketsField.Parent = listBucketsResponse
	nextPageTokenField.Parent = listBucketsResponse

	storageService := &api.Service{
		Name:        "Storage",
		ID:          ".google.storage.v2.Storage",
		Package:     "google.storage.v2",
		DefaultHost: "storage.googleapis.com",
		Methods: []*api.Method{
			{
				Name:         "CreateBucket",
				ID:           ".google.storage.v2.Storage.CreateBucket",
				InputTypeID:  ".google.storage.v2.CreateBucketRequest",
				InputType:    createBucketRequest,
				OutputTypeID: ".google.storage.v2.Bucket",
				OutputType:   bucket,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("buckets"),
						},
					},
				},
			},
			{
				Name:         "ListBuckets",
				ID:           ".google.storage.v2.Storage.ListBuckets",
				InputTypeID:  ".google.storage.v2.ListBucketsRequest",
				InputType:    listBucketsRequest,
				OutputTypeID: ".google.storage.v2.ListBucketsResponse",
				OutputType:   listBucketsResponse,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("buckets"),
						},
					},
				},
			},
		},
	}

	storageModel := api.NewTestAPI([]*api.Message{bucket, createBucketRequest, listBucketsRequest, listBucketsResponse}, nil, []*api.Service{storageService})
	storageModel.PackageName = "google.storage.v2"
	if err := api.CrossReference(storageModel); err != nil {
		t.Fatal(err)
	}
	api.UpdateMethodPagination(nil, storageModel)

	// StorageControl Messages & Service
	folder := &api.Message{
		Name:    "Folder",
		ID:      ".google.storage.control.v2.Folder",
		Package: "google.storage.control.v2",
	}
	createFolderRequest := &api.Message{
		Name:    "CreateFolderRequest",
		ID:      ".google.storage.control.v2.CreateFolderRequest",
		Package: "google.storage.control.v2",
		Fields: []*api.Field{
			{Name: "parent", JSONName: "parent", Typez: api.TypezString},
			{Name: "folder_id", JSONName: "folderId", Typez: api.TypezString},
		},
	}
	policy := &api.Message{
		Name:    "Policy",
		ID:      ".google.iam.v1.Policy",
		Package: "google.iam.v1",
	}
	getIamPolicyRequest := &api.Message{
		Name:    "GetIamPolicyRequest",
		ID:      ".google.iam.v1.GetIamPolicyRequest",
		Package: "google.iam.v1",
		Fields: []*api.Field{
			{Name: "resource", JSONName: "resource", Typez: api.TypezString},
		},
	}

	controlService := &api.Service{
		Name:        "StorageControl",
		ID:          ".google.storage.control.v2.StorageControl",
		Package:     "google.storage.control.v2",
		DefaultHost: "storage.googleapis.com",
		Methods: []*api.Method{
			{
				Name:         "CreateFolder",
				ID:           ".google.storage.control.v2.StorageControl.CreateFolder",
				InputTypeID:  ".google.storage.control.v2.CreateFolderRequest",
				InputType:    createFolderRequest,
				OutputTypeID: ".google.storage.control.v2.Folder",
				OutputType:   folder,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("parent").WithLiteral("folders"),
						},
					},
				},
			},
			{
				Name:         "GetIamPolicy",
				ID:           ".google.storage.control.v2.StorageControl.GetIamPolicy",
				InputTypeID:  ".google.iam.v1.GetIamPolicyRequest",
				InputType:    getIamPolicyRequest,
				OutputTypeID: ".google.iam.v1.Policy",
				OutputType:   policy,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("v2").WithVariableNamed("resource").WithLiteral(":getIamPolicy"),
						},
					},
				},
			},
		},
	}

	controlModel := api.NewTestAPI([]*api.Message{folder, createFolderRequest, policy, getIamPolicyRequest}, nil, []*api.Service{controlService})
	controlModel.PackageName = "google.storage.control.v2"
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
		{Name: "GoogleCloudGax", RequiredByServices: true},
		{Name: "GoogleCloudAuth", RequiredByServices: true},
		{Name: "GoogleIAMV1", ApiPackage: "google.iam.v1"},
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
	if !strings.Contains(protocolStr, "public protocol StorageControlProtocol {") {
		t.Errorf("StorageControlProtocol.swift missing StorageControlProtocol declaration:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "func createBucket(") ||
		!strings.Contains(protocolStr, "func listBuckets(") ||
		!strings.Contains(protocolStr, "func createFolder(") ||
		!strings.Contains(protocolStr, "func getIamPolicy(") {
		t.Errorf("StorageControlProtocol.swift missing unified methods:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "byItem: ListBucketsRequest, options: GoogleCloudGax.RequestOptions") ||
		!strings.Contains(protocolStr, "any AsyncSequence<Bucket, Swift.Error>") {
		t.Errorf("StorageControlProtocol.swift missing paginated helper method:\n%s", protocolStr)
	}
	if !strings.Contains(protocolStr, "import GoogleIAMV1") {
		t.Errorf("StorageControlProtocol.swift missing import GoogleIAMV1:\n%s", protocolStr)
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
	if !strings.Contains(clientStr, "public class StorageControlClient: StorageControlProtocol {") {
		t.Errorf("StorageControlClient.swift missing class declaration:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "private let storage: any Clients.StorageStub") ||
		!strings.Contains(clientStr, "private let control: any Clients.StorageControlStub") {
		t.Errorf("StorageControlClient.swift missing private stub fields:\n%s", clientStr)
	}
	if !strings.Contains(clientStr, "let sharedGrpcClient = try GoogleCloudGaxGRPC._GRPCClient(") ||
		!strings.Contains(clientStr, `withDefaultEndpoint: "https://storage.googleapis.com"`) {
		t.Errorf("StorageControlClient.swift missing shared _GRPCClient initialization:\n%s", clientStr)
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
	if !strings.Contains(clientStr, "byItem: ListBucketsRequest, options: GoogleCloudGax.RequestOptions") ||
		!strings.Contains(clientStr, "return GoogleCloudGax.PaginatedResponseSequence(listRpc: listRpc)") {
		t.Errorf("StorageControlClient.swift missing paginated sequence helper:\n%s", clientStr)
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
