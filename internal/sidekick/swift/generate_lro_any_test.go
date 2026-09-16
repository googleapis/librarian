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

// lroAnyTestModel builds a model shaped like the storage control API: a service
// with one long-running operation, plus the `Operation` message that carries
// the result.
func lroAnyTestModel(t *testing.T) *api.API {
	t.Helper()
	anyMessage := &api.Message{
		Name:    "Any",
		Package: "google.protobuf",
		ID:      ".google.protobuf.Any",
	}
	emptyMessage := &api.Message{
		Name:    "Empty",
		Package: "google.protobuf",
		ID:      ".google.protobuf.Empty",
	}
	metadata := &api.Message{
		Name:    "RenameFolderMetadata",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.RenameFolderMetadata",
	}
	folder := &api.Message{
		Name:    "Folder",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.Folder",
	}

	metadataField := &api.Field{
		Name:     "metadata",
		JSONName: "metadata",
		Typez:    api.TypezMessage,
		TypezID:  ".google.protobuf.Any",
		Optional: true,
	}
	responseField := &api.Field{
		Name:     "response",
		JSONName: "response",
		Typez:    api.TypezMessage,
		TypezID:  ".google.protobuf.Any",
		IsOneOf:  true,
	}
	resultGroup := &api.OneOf{
		Name:   "result",
		ID:     ".google.longrunning.Operation.result",
		Fields: []*api.Field{responseField},
	}
	operation := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
		Fields:  []*api.Field{metadataField, responseField},
		OneOfs:  []*api.OneOf{resultGroup},
	}
	metadataField.Parent = operation
	responseField.Parent = operation
	responseField.Group = resultGroup

	method := &api.Method{
		Name:         "RenameFolder",
		ID:           ".google.storage.control.v2.StorageControl.RenameFolder",
		InputTypeID:  ".google.storage.control.v2.Folder",
		OutputTypeID: ".google.longrunning.Operation",
		OperationInfo: &api.OperationInfo{
			MetadataTypeID: ".google.storage.control.v2.RenameFolderMetadata",
			ResponseTypeID: ".google.storage.control.v2.Folder",
		},
	}
	// A long-running operation can also resolve to a well-known type.
	deleteMethod := &api.Method{
		Name:         "DeleteFolderRecursive",
		ID:           ".google.storage.control.v2.StorageControl.DeleteFolderRecursive",
		InputTypeID:  ".google.storage.control.v2.Folder",
		OutputTypeID: ".google.longrunning.Operation",
		OperationInfo: &api.OperationInfo{
			MetadataTypeID: ".google.storage.control.v2.RenameFolderMetadata",
			ResponseTypeID: ".google.protobuf.Empty",
		},
	}
	service := &api.Service{
		Name:    "StorageControl",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.StorageControl",
		Methods: []*api.Method{method, deleteMethod},
	}

	model := api.NewTestAPI(
		[]*api.Message{operation, metadata, folder, anyMessage, emptyMessage},
		[]*api.Enum{},
		[]*api.Service{service})
	model.PackageName = "google.storage.control.v2"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	return model
}

// lroAnyTestLibrary mirrors the dependency configuration storage uses.
func lroAnyTestLibrary(t *testing.T, lroAnyConverter string) *config.Library {
	t.Helper()
	swiftPkg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleCloudGax", RequiredByServices: true},
		{Name: "GoogleAuth", RequiredByServices: true},
		{Name: "GoogleLongRunning", ApiPackage: "google.longrunning"},
		{Name: "GoogleCloudWKT", ApiPackage: "google.protobuf"},
	})
	swiftPkg.PackageNameOverride = "GoogleCloudStorage"
	swiftPkg.LibraryNameOverride = "GoogleCloudStorage"
	swiftPkg.LROAnyConverter = lroAnyConverter
	return &config.Library{
		Name:          "google-cloud-storage",
		CopyrightYear: "2026",
		Swift:         swiftPkg,
	}
}

// The `Any` fields of `Operation` route through the configured converter, so
// conversion never depends on SwiftProtobuf's global type registry.
func TestGenerateConversions_LROAnyConverter(t *testing.T) {
	outDir := t.TempDir()
	model := lroAnyTestModel(t)
	module := &config.SwiftModule{
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "StorageControlLROAnyConverter")

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Operation+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	got := extractBlock(t, gotContent, "  internal init(proto: ProtoType) throws {", "\n  }")
	wantInit := `  internal init(proto: ProtoType) throws {
    self.init()
    self.metadata = proto.hasMetadata ? try StorageControlLROAnyConverter.fromProto(proto.metadata) : nil
    if let oneof = proto.result {
      switch oneof {
      case .response(let value):
        self.result = .response(try StorageControlLROAnyConverter.fromProto(value))
      }
    }
    self._unknownFields.proto = proto.unknownFields.data
  }`
	if diff := cmp.Diff(wantInit, got); diff != "" {
		t.Errorf("init(proto:) mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, gotContent, "  internal func toProto() throws -> ProtoType {", "\n  }")
	wantToProto := `  internal func toProto() throws -> ProtoType {
    var proto = ProtoType()
    if let metadata = self.metadata { proto.metadata = try StorageControlLROAnyConverter.toProto(metadata) }
    if let oneof = self.result {
      switch oneof {
      case .response(let value):
        if let value = value {
          proto.result = .response(try StorageControlLROAnyConverter.toProto(value))
        }
      }
    }
    if !self._unknownFields.proto.isEmpty {
      try proto.merge(serializedBytes: self._unknownFields.proto)
    }
    return proto
  }`
	if diff := cmp.Diff(wantToProto, got); diff != "" {
		t.Errorf("toProto() mismatch (-want +got):\n%s", diff)
	}
}

// Without the setting, `Any` fields keep the generic conversion. This is the
// behavior for every library that does not convert long-running operations.
func TestGenerateConversions_LROAnyConverterUnset(t *testing.T) {
	outDir := t.TempDir()
	model := lroAnyTestModel(t)
	module := &config.SwiftModule{
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "")

	if err := GenerateConversions(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "Operation+Convert.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	if strings.Contains(gotContent, "LROAnyConverter") {
		t.Errorf("expected no converter call in generated file, got:\n%s", gotContent)
	}
	if !strings.Contains(gotContent, "try .init(proto: proto.metadata)") {
		t.Errorf("expected generic conversion for metadata, got:\n%s", gotContent)
	}
}

// The converter covers the metadata and response types the service declares,
// and nothing else.
func TestGenerateStubs_LROAnyConverter(t *testing.T) {
	outDir := t.TempDir()
	model := lroAnyTestModel(t)
	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "StorageControlLROAnyConverter")

	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(outDir, "StorageControl+LROAnyConverter.swift"))
	if err != nil {
		t.Fatal(err)
	}
	gotContent := string(b)

	for _, want := range []string{
		"internal enum StorageControlLROAnyConverter {",
		`case "type.googleapis.com/google.storage.control.v2.Folder":`,
		`case "type.googleapis.com/google.storage.control.v2.RenameFolderMetadata":`,
		`case "type.googleapis.com/google.protobuf.Empty":`,
		"proto: SwiftProtobuf.Google_Protobuf_Empty(serializedBytes: proto.value)",
		"throw ProtobufConversionError.unknownTypeUrl(typeUrl: proto.typeURL)",
		"throw ProtobufConversionError.unknownTypeUrl(typeUrl: any.typeUrl)",
	} {
		if !strings.Contains(gotContent, want) {
			t.Errorf("expected generated converter to contain %q, got:\n%s", want, gotContent)
		}
	}
}

// A library that does not convert long-running operations gets no converter,
// even though its services declare the payload types one would cover.
func TestGenerateStubs_LROAnyConverterUnset(t *testing.T) {
	outDir := t.TempDir()
	model := lroAnyTestModel(t)
	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "")

	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(outDir, "StorageControl+LROAnyConverter.swift")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no converter file at %s", path)
	}
}

// Services without long-running operations get no converter, so the generated
// package does not carry an unused type.
func TestGenerateStubs_LROAnyConverterOmitted(t *testing.T) {
	outDir := t.TempDir()
	method := &api.Method{
		Name:         "GetFolder",
		ID:           ".google.storage.control.v2.StorageControl.GetFolder",
		InputTypeID:  ".google.storage.control.v2.Folder",
		OutputTypeID: ".google.storage.control.v2.Folder",
	}
	folder := &api.Message{
		Name:    "Folder",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.Folder",
	}
	service := &api.Service{
		Name:    "StorageControl",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.StorageControl",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{folder}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "google.storage.control.v2"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "StorageControlLROAnyConverter")
	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(outDir, "StorageControl+LROAnyConverter.swift")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no converter file at %s", path)
	}
}
