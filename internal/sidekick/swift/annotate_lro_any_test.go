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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// lroAnyFieldsTestModel builds a model whose `Operation` carries the fields the
// real one does: the two `Any` payloads next to a scalar, a boolean, and a
// message that is not an `Any`. `Status` carries `Any` values of its own, and
// stands in for the messages that keep the generic conversion.
func lroAnyFieldsTestModel(t *testing.T) *api.API {
	t.Helper()
	anyMessage := &api.Message{
		Name:    "Any",
		Package: "google.protobuf",
		ID:      ".google.protobuf.Any",
	}
	detailsField := &api.Field{
		Name:     "details",
		JSONName: "details",
		Typez:    api.TypezMessage,
		TypezID:  ".google.protobuf.Any",
		Repeated: true,
	}
	status := &api.Message{
		Name:    "Status",
		Package: "google.rpc",
		ID:      ".google.rpc.Status",
		Fields:  []*api.Field{detailsField},
	}
	detailsField.Parent = status

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

	nameField := &api.Field{
		Name:     "name",
		JSONName: "name",
		Typez:    api.TypezString,
	}
	metadataField := &api.Field{
		Name:     "metadata",
		JSONName: "metadata",
		Typez:    api.TypezMessage,
		TypezID:  ".google.protobuf.Any",
		Optional: true,
	}
	doneField := &api.Field{
		Name:     "done",
		JSONName: "done",
		Typez:    api.TypezBool,
	}
	errorField := &api.Field{
		Name:     "error",
		JSONName: "error",
		Typez:    api.TypezMessage,
		TypezID:  ".google.rpc.Status",
		IsOneOf:  true,
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
		Fields: []*api.Field{errorField, responseField},
	}
	operation := &api.Message{
		Name:    "Operation",
		Package: "google.longrunning",
		ID:      ".google.longrunning.Operation",
		Fields:  []*api.Field{nameField, metadataField, doneField, errorField, responseField},
		OneOfs:  []*api.OneOf{resultGroup},
	}
	for _, field := range operation.Fields {
		field.Parent = operation
	}
	errorField.Group = resultGroup
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
	service := &api.Service{
		Name:    "StorageControl",
		Package: "google.storage.control.v2",
		ID:      ".google.storage.control.v2.StorageControl",
		Methods: []*api.Method{method},
	}

	model := api.NewTestAPI(
		[]*api.Message{operation, status, metadata, folder, anyMessage},
		[]*api.Enum{},
		[]*api.Service{service})
	model.PackageName = "google.storage.control.v2"
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	return model
}

// Only the `Any` fields of `Operation` route through the converter. The rest of
// `Operation`, and the `Any` fields of every other message, keep the conversion
// they would have without the setting: their payload types are not declared
// anywhere the generator can see, so there is no table to dispatch on.
func TestAnnotateLROAnyFields_OnlyOperationAnyFields(t *testing.T) {
	model := lroAnyFieldsTestModel(t)
	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "StorageControlLROAnyConverter")

	c, err := newCodec(model, library, module, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, id := range []string{".google.longrunning.Operation", ".google.rpc.Status"} {
		message := model.Message(id)
		if message == nil {
			t.Fatalf("message %q not found in the test model", id)
		}
		for _, field := range message.Fields {
			annotations, ok := field.Codec.(*fieldAnnotations)
			if !ok {
				t.Fatalf("field %q of %q is missing its Swift annotations", field.Name, id)
			}
			if annotations.LROAnyConverter == "" {
				continue
			}
			if annotations.LROAnyConverter != "StorageControlLROAnyConverter" {
				t.Errorf("field %q of %q names converter %q, want %q",
					field.Name, id, annotations.LROAnyConverter, "StorageControlLROAnyConverter")
			}
			got = append(got, fmt.Sprintf("%s.%s", message.Name, field.Name))
		}
	}
	want := []string{"Operation.metadata", "Operation.response"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("converted fields mismatch (-want +got):\n%s", diff)
	}
}

// A payload type that this module does not generate is left out of the table
// rather than generating a call to a type that does not exist. At runtime it
// takes the converter's generic path.
//
// The table is checked directly: the generator annotates every message a
// method refers to, so within a single model there is no way to shape an
// unannotated payload type.
func TestLROAnyTypes_PayloadTypeNotGenerated(t *testing.T) {
	model := lroAnyTestModel(t)
	module := &config.SwiftModule{
		ModuleType: "grpc-client",
		ModulePath: "StorageControlProtos",
	}
	library := lroAnyTestLibrary(t, "StorageControlLROAnyConverter")

	c, err := newCodec(model, library, module, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.annotateModel(); err != nil {
		t.Fatal(err)
	}
	// Stand in for a payload type another module owns.
	model.Message(".google.storage.control.v2.Folder").Codec = nil

	types, err := c.lroAnyTypes(model.Services[0].Methods)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, payload := range types {
		got = append(got, payload.TypeURL)
	}
	want := []string{
		"type.googleapis.com/google.protobuf.Empty",
		"type.googleapis.com/google.storage.control.v2.RenameFolderMetadata",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("payload types mismatch (-want +got):\n%s", diff)
	}
}
