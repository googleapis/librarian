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

package cpp

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod_Unary(t *testing.T) {
	req := api.NewTestMessage("GetItemRequest")
	resp := api.NewTestMessage("Item")
	method := api.NewTestMethod("GetItem").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got, ok := method.Codec.(*methodAnnotations)
	if !ok {
		t.Fatalf("expected *methodAnnotations, got %T", method.Codec)
	}

	want := &methodAnnotations{
		ServiceName:              "ItemService",
		Name:                     "GetItem",
		CppRequestType:           "test::GetItemRequest",
		CppResponseType:          "test::Item",
		CppReturnType:            "StatusOr<test::Item>",
		IsUnary:                  true,
		LongrunningOperationType: "google::longrunning::Operation",
	}

	diff := cmp.Diff(want, got,
		cmpopts.IgnoreFields(methodAnnotations{}, "Service", "Comments", "RequestComments", "NoAwaitComments", "OperationComments", "DocLines"))
	if diff != "" {
		t.Logf("methodAnnotations")
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if len(got.DocLines) == 0 {
		t.Errorf("expected non-empty DocLines")
	}
	if !strings.Contains(got.RequestComments, "GetItem") {
		t.Errorf("expected RequestComments to contain GetItem, got: %s", got.RequestComments)
	}
}

func TestAnnotateMethod_EmptyResponse(t *testing.T) {
	req := api.NewTestMessage("DeleteItemRequest")
	empty := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	method := api.NewTestMethod("DeleteItem").WithInput(req).WithOutput(empty)
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, empty}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	if got.CppReturnType != "Status" {
		t.Errorf("expected CppReturnType Status, got: %s", got.CppReturnType)
	}
	if !got.IsResponseTypeEmpty {
		t.Errorf("expected IsResponseTypeEmpty true, got false")
	}
}

func TestAnnotateMethod_Paginated(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("test")
	itemField := api.NewTestField("items").WithMessageType(itemMsg).WithRepeated()
	respMsg := api.NewTestMessage("ListItemsResponse").WithFields(
		itemField,
		api.NewTestField("next_page_token").WithType(api.TypezString),
	)
	respMsg.Pagination = &api.PaginationInfo{
		PageableItem: itemField,
	}

	reqMsg := api.NewTestMessage("ListItemsRequest").WithFields(
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("page_token").WithType(api.TypezString),
	)

	method := api.NewTestMethod("ListItems").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithPagination(api.NewTestField("page_token").WithType(api.TypezString))
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, itemMsg}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	if !got.IsPaginated {
		t.Errorf("expected IsPaginated true, got false")
	}
	if got.RangeOutputType != "test::Item" {
		t.Errorf("expected RangeOutputType test::Item, got %s", got.RangeOutputType)
	}
	if got.IsUnary {
		t.Errorf("expected IsUnary false for paginated method, got true")
	}
}

func TestAnnotateMethod_LRO(t *testing.T) {
	req := api.NewTestMessage("CreateItemRequest")
	op := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	item := api.NewTestMessage("Item")
	meta := api.NewTestMessage("CreateItemMetadata")

	method := api.NewTestMethod("CreateItem").
		WithInput(req).
		WithOutput(op).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: item.ID,
			MetadataTypeID: meta.ID,
		})

	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, op, item, meta}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	if !got.IsLongrunning {
		t.Errorf("expected IsLongrunning true, got false")
	}
	if got.LongrunningDeducedResponseType != "test::Item" {
		t.Errorf("expected deduced response test::Item, got %s", got.LongrunningDeducedResponseType)
	}
	if got.CppReturnType != "StatusOr<test::Item>" {
		t.Errorf("expected CppReturnType StatusOr<test::Item>, got %s", got.CppReturnType)
	}
	if got.NoAwaitComments == "" {
		t.Errorf("expected non-empty NoAwaitComments")
	}
	if got.OperationComments == "" {
		t.Errorf("expected non-empty OperationComments")
	}
}

func TestAnnotateMethod_Streaming(t *testing.T) {
	req := api.NewTestMessage("StreamRequest")
	resp := api.NewTestMessage("StreamResponse")

	readMethod := api.NewTestMethod("StreamRead").WithInput(req).WithOutput(resp).WithServerSideStreaming()
	writeMethod := api.NewTestMethod("StreamWrite").WithInput(req).WithOutput(resp).WithClientSideStreaming()
	bidiMethod := api.NewTestMethod("StreamBidi").WithInput(req).WithOutput(resp).WithBidiStreaming()

	svc := api.NewTestService("StreamService").WithMethods(readMethod, writeMethod, bidiMethod)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	gotRead := readMethod.Codec.(*methodAnnotations)
	if !gotRead.IsStreamingRead || gotRead.IsStreamingWrite || gotRead.IsBidirStreaming {
		t.Errorf("unexpected streaming flags for read: %+v", gotRead)
	}

	gotWrite := writeMethod.Codec.(*methodAnnotations)
	if !gotWrite.IsStreamingWrite || gotWrite.IsStreamingRead || gotWrite.IsBidirStreaming {
		t.Errorf("unexpected streaming flags for write: %+v", gotWrite)
	}

	gotBidi := bidiMethod.Codec.(*methodAnnotations)
	if !gotBidi.IsBidirStreaming || gotBidi.IsStreamingRead || gotBidi.IsStreamingWrite {
		t.Errorf("unexpected streaming flags for bidi: %+v", gotBidi)
	}
}

func TestAnnotateMethod_Signatures(t *testing.T) {
	nameField := api.NewTestField("name").WithType(api.TypezString)
	exportField := api.NewTestField("export").WithType(api.TypezBool) // C++ keyword
	req := api.NewTestMessage("InspectRequest").WithFields(nameField, exportField)
	resp := api.NewTestMessage("InspectResponse")

	method := api.NewTestMethod("Inspect").
		WithInput(req).
		WithOutput(resp).
		WithSignatures(&api.MethodSignature{
			Names: []string{"name", "export"},
		})

	svc := api.NewTestService("InspectService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	if len(got.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(got.Signatures))
	}

	sig := got.Signatures[0]
	wantParams := []*parameterAnnotation{
		{Type: "std::string const&", Name: "name"},
		{Type: "bool", Name: "export_"},
	}
	if diff := cmp.Diff(wantParams, sig.Parameters); diff != "" {
		t.Errorf("parameters mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_Async(t *testing.T) {
	req := api.NewTestMessage("GetItemRequest")
	resp := api.NewTestMessage("Item")
	method := api.NewTestMethod("GetItem").WithInput(req).WithOutput(resp)
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

	t.Run("method in GenAsyncRPCs", func(t *testing.T) {
		c := newCodec(&config.CppLibrary{
			GenAsyncRPCs: []string{"GetItem"},
		})
		if err := c.annotateModel(model); err != nil {
			t.Fatal(err)
		}
		got, ok := method.Codec.(*methodAnnotations)
		if !ok {
			t.Fatalf("expected *methodAnnotations, got %T", method.Codec)
		}
		if !got.IsAsync {
			t.Errorf("expected IsAsync true for method in GenAsyncRPCs, got false")
		}
	})

	t.Run("method not in GenAsyncRPCs", func(t *testing.T) {
		cSync := newCodec(nil)
		if err := cSync.annotateModel(model); err != nil {
			t.Fatal(err)
		}
		gotSync := method.Codec.(*methodAnnotations)
		if gotSync.IsAsync {
			t.Errorf("expected IsAsync false for method not in GenAsyncRPCs, got true")
		}
	})

	t.Run("LRO method", func(t *testing.T) {
		op := api.NewTestMessage("Operation").WithPackage("google.longrunning")
		lroMethod := api.NewTestMethod("CreateItem").
			WithInput(req).
			WithOutput(op).
			WithOperationInfo(&api.OperationInfo{
				ResponseTypeID: resp.ID,
			})
		lroSvc := api.NewTestService("LroService").WithMethods(lroMethod)
		lroModel := api.NewTestAPI([]*api.Message{req, resp, op}, nil, []*api.Service{lroSvc})
		cLro := newCodec(nil)
		if err := cLro.annotateModel(lroModel); err != nil {
			t.Fatal(err)
		}
		gotLro := lroMethod.Codec.(*methodAnnotations)
		if !gotLro.IsAsync {
			t.Errorf("expected IsAsync true for LRO method, got false")
		}
	})
}
