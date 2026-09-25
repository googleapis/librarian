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
	anyMessage := api.NewTestMessage("Any").WithPackage("google.protobuf")
	detailsField := api.NewTestField("details").
		WithMessageType(anyMessage).
		WithRepeated()
	status := api.NewTestMessage("Status").
		WithPackage("google.rpc").
		WithFields(detailsField)

	metadata := api.NewTestMessage("RenameFolderMetadata").
		WithPackage("google.storage.control.v2")
	folder := api.NewTestMessage("Folder").
		WithPackage("google.storage.control.v2")

	nameField := api.NewTestField("name").WithType(api.TypezString)
	metadataField := api.NewTestField("metadata").
		WithMessageType(anyMessage).
		WithOptional()
	doneField := api.NewTestField("done").WithType(api.TypezBool)
	errorField := api.NewTestField("error").WithMessageType(status)
	responseField := api.NewTestField("response").WithMessageType(anyMessage)
	resultGroup := api.NewTestOneOf("result").WithFields(errorField, responseField)
	operation := api.NewTestMessage("Operation").
		WithPackage("google.longrunning").
		WithFields(nameField, metadataField, doneField).
		WithOneOfs(resultGroup)

	method := api.NewTestMethod("RenameFolder").
		WithInput(folder).
		WithOutput(operation).
		WithPathInfo(nil).
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: metadata.ID,
			ResponseTypeID: folder.ID,
		})
	service := api.NewTestService("StorageControl").
		WithPackage("google.storage.control.v2").
		WithMethods(method)

	model := api.NewTestAPI(
		[]*api.Message{operation, status, metadata, folder, anyMessage},
		nil,
		[]*api.Service{service}).
		WithPackageName("google.storage.control.v2")
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
