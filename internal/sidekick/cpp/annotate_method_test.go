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
	"slices"
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
		StubMemberName:           "grpc_stub_",
		LongrunningOperationType: "google::longrunning::Operation",
		Idempotency:              "kNonIdempotent",
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
	type emptyResult struct {
		CppReturnType       string
		IsResponseTypeEmpty bool
	}
	want := emptyResult{
		CppReturnType:       "Status",
		IsResponseTypeEmpty: true,
	}
	gotResult := emptyResult{
		CppReturnType:       got.CppReturnType,
		IsResponseTypeEmpty: got.IsResponseTypeEmpty,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_Paginated(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("test")
	itemField := api.NewTestField("items").WithMessageType(itemMsg).WithRepeated()
	nextPageToken := api.NewTestField("next_page_token").WithType(api.TypezString)
	respMsg := api.NewTestMessage("ListItemsResponse").
		WithPagination(nextPageToken, itemField)

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
	type paginationResult struct {
		IsPaginated     bool
		RangeOutputType string
		IsUnary         bool
	}
	want := paginationResult{
		IsPaginated:     true,
		RangeOutputType: "test::Item",
		IsUnary:         false,
	}
	gotResult := paginationResult{
		IsPaginated:     got.IsPaginated,
		RangeOutputType: got.RangeOutputType,
		IsUnary:         got.IsUnary,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
	type lroResult struct {
		IsLongrunning                  bool
		LongrunningDeducedResponseType string
		CppReturnType                  string
	}
	want := lroResult{
		IsLongrunning:                  true,
		LongrunningDeducedResponseType: "test::Item",
		CppReturnType:                  "StatusOr<test::Item>",
	}
	gotResult := lroResult{
		IsLongrunning:                  got.IsLongrunning,
		LongrunningDeducedResponseType: got.LongrunningDeducedResponseType,
		CppReturnType:                  got.CppReturnType,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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

	type streamingFlags struct {
		IsStreamingRead  bool
		IsStreamingWrite bool
		IsBidirStreaming bool
	}
	gotRead := readMethod.Codec.(*methodAnnotations)
	if diff := cmp.Diff(streamingFlags{IsStreamingRead: true}, streamingFlags{
		IsStreamingRead:  gotRead.IsStreamingRead,
		IsStreamingWrite: gotRead.IsStreamingWrite,
		IsBidirStreaming: gotRead.IsBidirStreaming,
	}); diff != "" {
		t.Errorf("read mismatch (-want +got):\n%s", diff)
	}

	gotWrite := writeMethod.Codec.(*methodAnnotations)
	if diff := cmp.Diff(streamingFlags{IsStreamingWrite: true}, streamingFlags{
		IsStreamingRead:  gotWrite.IsStreamingRead,
		IsStreamingWrite: gotWrite.IsStreamingWrite,
		IsBidirStreaming: gotWrite.IsBidirStreaming,
	}); diff != "" {
		t.Errorf("write mismatch (-want +got):\n%s", diff)
	}

	gotBidi := bidiMethod.Codec.(*methodAnnotations)
	if diff := cmp.Diff(streamingFlags{IsBidirStreaming: true}, streamingFlags{
		IsStreamingRead:  gotBidi.IsStreamingRead,
		IsStreamingWrite: gotBidi.IsStreamingWrite,
		IsBidirStreaming: gotBidi.IsBidirStreaming,
	}); diff != "" {
		t.Errorf("bidi mismatch (-want +got):\n%s", diff)
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
		{Type: "std::string const&", Name: "name", FieldName: "name", IsScalar: true},
		{Type: "bool", Name: "export_", FieldName: "export_", IsScalar: true},
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

func TestAnnotateMethod_Idempotency(t *testing.T) {
	req := api.NewTestMessage("Request")
	resp := api.NewTestMessage("Response")

	getMethod := api.NewTestMethod("GetThing").WithInput(req).WithOutput(resp).WithVerb("GET")
	putMethod := api.NewTestMethod("PutThing").WithInput(req).WithOutput(resp).WithVerb("PUT")
	postMethod := api.NewTestMethod("PostThing").WithInput(req).WithOutput(resp).WithVerb("POST")
	postOverride := api.NewTestMethod("PostThing").WithInput(req).WithOutput(resp).WithVerb("POST")

	for _, test := range []struct {
		name   string
		method *api.Method
		cfg    *config.CppLibrary
		want   string
	}{
		{
			name:   "default fallback is kNonIdempotent",
			method: api.NewTestMethod("DoThing").WithInput(req).WithOutput(resp),
			want:   "kNonIdempotent",
		},
		{
			name:   "GET verb is kIdempotent",
			method: getMethod,
			want:   "kIdempotent",
		},
		{
			name:   "PUT verb is kIdempotent",
			method: putMethod,
			want:   "kIdempotent",
		},
		{
			name:   "POST verb is kNonIdempotent",
			method: postMethod,
			want:   "kNonIdempotent",
		},
		{
			name:   "config override takes precedence",
			method: postOverride,
			cfg: &config.CppLibrary{
				IdempotencyOverrides: []config.IdempotencyRule{
					{RPCName: "PostThing", Idempotency: "kIdempotent"},
				},
			},
			want: "kIdempotent",
		},
		{
			name: "GetIamPolicy is kIdempotent",
			method: api.NewTestMethod("GetIamPolicy").
				WithInput(api.NewTestMessage("GetIamPolicyRequest").WithPackage("google.iam.v1")).
				WithOutput(api.NewTestMessage("Policy").WithPackage("google.iam.v1")),
			want: "kIdempotent",
		},
		{
			name: "TestIamPermissions is kIdempotent",
			method: api.NewTestMethod("TestIamPermissions").
				WithInput(api.NewTestMessage("TestIamPermissionsRequest").WithPackage("google.iam.v1")).
				WithOutput(api.NewTestMessage("TestIamPermissionsResponse").WithPackage("google.iam.v1")),
			want: "kIdempotent",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("TestService").WithMethods(test.method)
			model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
			c := newCodec(test.cfg)
			if err := c.annotateModel(model); err != nil {
				t.Fatal(err)
			}
			got := test.method.Codec.(*methodAnnotations)
			if diff := cmp.Diff(test.want, got.Idempotency); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_Routing(t *testing.T) {
	req := api.NewTestMessage("Request")
	resp := api.NewTestMessage("Response")

	t.Run("simple routing", func(t *testing.T) {
		method := api.NewTestMethod("RouteMethod").WithInput(req).WithOutput(resp)
		method.Routing = []*api.RoutingInfo{
			{
				Name: "table_name",
				Variants: []*api.RoutingInfoVariant{
					{
						FieldPath: []string{"table_name"},
						Matching:  api.RoutingPathSpec{Segments: []string{"*"}},
					},
				},
			},
		}
		svc := api.NewTestService("RouteService").WithMethods(method)
		model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
		c := newCodec(nil)
		if err := c.annotateModel(model); err != nil {
			t.Fatal(err)
		}
		got := method.Codec.(*methodAnnotations)
		if !got.HasRouting {
			t.Fatalf("expected HasRouting to be true")
		}
		if got.RoutingParamsCount != 1 {
			t.Errorf("expected 1 routing param, got %d", got.RoutingParamsCount)
		}
		if len(got.RoutingParamMatchers) != 1 {
			t.Fatalf("expected 1 matcher, got %d", len(got.RoutingParamMatchers))
		}
		matcher := got.RoutingParamMatchers[0]
		if !matcher.IsSimple {
			t.Errorf("expected matcher to be simple")
		}
		if matcher.ParamKey != "table_name" {
			t.Errorf("expected ParamKey 'table_name', got %q", matcher.ParamKey)
		}
	})

	t.Run("complex pattern routing", func(t *testing.T) {
		method := api.NewTestMethod("RouteMethod").WithInput(req).WithOutput(resp)
		method.Routing = []*api.RoutingInfo{
			{
				Name: "parent",
				Variants: []*api.RoutingInfoVariant{
					{
						FieldPath: []string{"parent"},
						Prefix:    api.RoutingPathSpec{Segments: []string{"projects", "*"}},
						Matching:  api.RoutingPathSpec{Segments: []string{"instances", "*"}},
					},
				},
			},
		}
		svc := api.NewTestService("RouteService").WithMethods(method)
		model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
		c := newCodec(nil)
		if err := c.annotateModel(model); err != nil {
			t.Fatal(err)
		}
		got := method.Codec.(*methodAnnotations)
		if !got.HasRouting {
			t.Fatalf("expected HasRouting to be true")
		}
		matcher := got.RoutingParamMatchers[0]
		if matcher.IsSimple {
			t.Errorf("expected complex matcher to have IsSimple=false")
		}
		if len(matcher.Patterns) != 1 {
			t.Fatalf("expected 1 pattern, got %d", len(matcher.Patterns))
		}
		if !matcher.Patterns[0].HasRegex {
			t.Errorf("expected HasRegex=true for complex pattern")
		}
	})
}

func TestAnnotateMethod_AutoPopulatedRequestId(t *testing.T) {
	req := api.NewTestMessage("Request")
	resp := api.NewTestMessage("Response")
	method := api.NewTestMethod("DoOp").WithInput(req).WithOutput(resp)
	method.AutoPopulated = []*api.Field{api.NewTestField("request_id")}
	svc := api.NewTestService("OpService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})
	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	got := method.Codec.(*methodAnnotations)
	type requestIdResult struct {
		HasRequestId       bool
		RequestIdFieldName string
	}
	want := requestIdResult{
		HasRequestId:       true,
		RequestIdFieldName: "request_id",
	}
	gotResult := requestIdResult{
		HasRequestId:       got.HasRequestId,
		RequestIdFieldName: got.RequestIdFieldName,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_ParameterAnnotationTypes(t *testing.T) {
	subMsg := api.NewTestMessage("SubMsg")
	mapField := api.NewTestField("labels").WithMap()
	repField := api.NewTestField("tags").WithRepeated()
	msgField := api.NewTestField("sub").WithMessageType(subMsg)
	scalarField := api.NewTestField("count").WithType(api.TypezInt32)

	req := api.NewTestMessage("Request").WithFields(mapField, repField, msgField, scalarField)
	resp := api.NewTestMessage("Response")
	method := api.NewTestMethod("DoOp").WithInput(req).WithOutput(resp).
		WithSignatures(&api.MethodSignature{
			Names: []string{"labels", "tags", "sub", "count"},
		})
	svc := api.NewTestService("OpService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{subMsg, req, resp}, nil, []*api.Service{svc})
	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	got := method.Codec.(*methodAnnotations)
	if len(got.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(got.Signatures))
	}
	params := got.Signatures[0].Parameters
	if len(params) != 4 {
		t.Fatalf("expected 4 parameters, got %d", len(params))
	}
	type paramFlags struct {
		IsMap           bool
		IsRepeated      bool
		IsMapOrRepeated bool
		IsMessage       bool
		IsScalar        bool
	}
	wantParams := []paramFlags{
		{IsMap: true, IsMapOrRepeated: true},
		{IsRepeated: true, IsMapOrRepeated: true},
		{IsMessage: true},
		{IsScalar: true},
	}
	var gotParams []paramFlags
	for _, p := range params {
		gotParams = append(gotParams, paramFlags{
			IsMap:           p.IsMap,
			IsRepeated:      p.IsRepeated,
			IsMapOrRepeated: p.IsMapOrRepeated,
			IsMessage:       p.IsMessage,
			IsScalar:        p.IsScalar,
		})
	}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_Rest(t *testing.T) {
	pageSizeField := api.NewTestField("page_size").WithType(api.TypezInt32)
	filterField := api.NewTestField("filter").WithType(api.TypezString)
	boolField := api.NewTestField("show_deleted").WithType(api.TypezBool)
	nameField := api.NewTestField("name").WithType(api.TypezString)
	req := api.NewTestMessage("GetItemRequest").WithFields(nameField, pageSizeField, filterField, boolField)
	resp := api.NewTestMessage("Item")

	pt := (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("items").WithVariableNamed("name").WithVerb("cancel")
	binding := &api.PathBinding{
		Verb:         "GET",
		PathTemplate: pt,
		QueryParameters: map[string]bool{
			"page_size":    true,
			"filter":       true,
			"show_deleted": true,
		},
	}
	method := api.NewTestMethod("GetItem").
		WithInput(req).
		WithOutput(resp)
	method.PathInfo = &api.PathInfo{
		Bindings:      []*api.PathBinding{binding},
		BodyFieldPath: "*",
	}

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

	type restMethodSummary struct {
		HasRestPath             bool
		RestVerb                string
		HasRestPathVerb         bool
		RestPathVerb            string
		RestHasQueryParams      bool
		RestRequestBodyAccessor string
		RestReturnTypeName      string
		RestPayloadType         string
		IsRestRpc               bool
	}
	wantSummary := restMethodSummary{
		HasRestPath:             true,
		RestVerb:                "Get",
		HasRestPathVerb:         true,
		RestPathVerb:            "cancel",
		RestHasQueryParams:      true,
		RestRequestBodyAccessor: "request",
		RestReturnTypeName:      "StatusOr<test::Item>",
		RestPayloadType:         "test::Item",
		IsRestRpc:               true,
	}
	gotSummary := restMethodSummary{
		HasRestPath:             got.HasRestPath,
		RestVerb:                got.RestVerb,
		HasRestPathVerb:         got.HasRestPathVerb,
		RestPathVerb:            got.RestPathVerb,
		RestHasQueryParams:      got.RestHasQueryParams,
		RestRequestBodyAccessor: got.RestRequestBodyAccessor,
		RestReturnTypeName:      got.RestReturnTypeName,
		RestPayloadType:         got.RestPayloadType,
		IsRestRpc:               got.IsRestRpc,
	}
	if diff := cmp.Diff(wantSummary, gotSummary); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantSegments := []*restPathSegmentAnnotation{
		{IsApiVersion: true, ApiVersion: "v1", HasNext: true},
		{IsLiteral: true, Literal: "items", HasNext: true},
		{IsField: true, FieldAccessor: "name()", HasNext: false},
	}
	if diff := cmp.Diff(wantSegments, got.RestPathSegments); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantQueryParams := []*queryParamAnnotation{
		{ParamKey: "filter", FieldAccessor: "filter()", IsString: true},
		{ParamKey: "page_size", FieldAccessor: "page_size()", IsNumber: true},
		{ParamKey: "show_deleted", FieldAccessor: "show_deleted()", IsBool: true},
	}
	if diff := cmp.Diff(wantQueryParams, got.RestQueryParams, cmpopts.SortSlices(func(a, b *queryParamAnnotation) bool {
		return a.ParamKey < b.ParamKey
	})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_HttpRoutingParams(t *testing.T) {
	req := api.NewTestMessage("GetDatabaseRequest").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
	)
	pt := (&api.PathTemplate{}).
		WithLiteral("v1").
		WithVariable(api.NewPathVariable("name"))
	binding := &api.PathBinding{
		Verb:         "GET",
		PathTemplate: pt,
	}
	method := api.NewTestMethod("GetDatabase").
		WithInput(req).
		WithOutput(api.NewTestMessage("Database"))
	method.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{binding},
	}
	svc := api.NewTestService("DatabaseAdmin").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	if !got.HasHttpRouting {
		t.Errorf("expected HasHttpRouting to be true")
	}
	wantParams := []*httpRoutingParamAnnotation{
		{Key: "name", FieldAccessor: "name()", HasNext: false},
	}
	if diff := cmp.Diff(wantParams, got.HttpRoutingParams); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_StubMemberNames(t *testing.T) {
	for _, test := range []struct {
		name            string
		sourceServiceID string
		want            string
	}{
		{name: "locations service", sourceServiceID: "google.cloud.location.Locations", want: "locations_stub_"},
		{name: "iam service", sourceServiceID: "google.iam.v1.IAMPolicy", want: "iampolicy_stub_"},
		{name: "operations service", sourceServiceID: "google.longrunning.Operations", want: "operations_stub_"},
		{name: "custom service", sourceServiceID: "google.example.v1.EchoService", want: "grpc_stub_"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := api.NewTestMessage("Req")
			resp := api.NewTestMessage("Resp")
			method := api.NewTestMethod("Method").WithInput(req).WithOutput(resp)
			method.SourceServiceID = test.sourceServiceID
			svc := api.NewTestService("TestService").WithMethods(method)
			model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

			c := newCodec(nil)
			if err := c.annotateModel(model); err != nil {
				t.Fatal(err)
			}
			got := method.Codec.(*methodAnnotations)
			if diff := cmp.Diff(test.want, got.StubMemberName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateMethod_DeprecatedSignatureFields(t *testing.T) {
	depField := api.NewTestField("deprecated_param").WithType(api.TypezString)
	depField.Deprecated = true
	normalField := api.NewTestField("normal_param").WithType(api.TypezString)
	req := api.NewTestMessage("Req").WithFields(depField, normalField)
	resp := api.NewTestMessage("Resp")

	method := api.NewTestMethod("MethodWithDep").
		WithInput(req).
		WithOutput(resp).
		WithSignatures(&api.MethodSignature{
			Fields: []*api.Field{normalField, depField},
		})
	svc := api.NewTestService("DepService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	got := method.Codec.(*methodAnnotations)
	if !got.HasDeprecatedFieldInSignature {
		t.Errorf("expected HasDeprecatedFieldInSignature to be true")
	}
}

func TestAnnotateMethod_LongrunningResponseIsEmpty(t *testing.T) {
	emptyMsg := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	metaMsg := api.NewTestMessage("Meta").WithPackage("test")
	req := api.NewTestMessage("Req").WithPackage("test")
	opMsg := api.NewTestMessage("Operation").WithPackage("google.longrunning")

	method := api.NewTestMethod("DropDatabase").
		WithInput(req).
		WithOutput(opMsg).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: emptyMsg.ID,
			MetadataTypeID: metaMsg.ID,
		})
	svc := api.NewTestService("Admin").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{emptyMsg, metaMsg, req, opMsg}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	got := method.Codec.(*methodAnnotations)
	if !got.LongrunningResponseIsEmpty {
		t.Errorf("expected LongrunningResponseIsEmpty to be true")
	}
}

func TestAnnotateMethod_ListOperations_QueryParamFilter(t *testing.T) {
	reqMsg := api.NewTestMessage("ListOperationsRequest").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("return_partial_success").WithType(api.TypezBool),
	)
	respMsg := api.NewTestMessage("ListOperationsResponse")
	pt := (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")
	binding := &api.PathBinding{
		Verb:         "GET",
		PathTemplate: pt,
		QueryParameters: map[string]bool{
			"name":                   true,
			"page_size":              true,
			"return_partial_success": true,
		},
	}
	method := api.NewTestMethod("ListOperations").
		WithInput(reqMsg).
		WithOutput(respMsg)
	method.PathInfo = &api.PathInfo{
		Bindings: []*api.PathBinding{binding},
	}
	svc := api.NewTestService("Operations").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg}, nil, []*api.Service{svc})

	libCfg := &config.CppLibrary{GenerateRestTransport: true}
	c := newCodec(libCfg)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}
	got := method.Codec.(*methodAnnotations)
	if slices.ContainsFunc(got.RestQueryParams, func(qp *queryParamAnnotation) bool {
		return qp.ParamKey == "return_partial_success"
	}) {
		t.Errorf("expected return_partial_success to be filtered out of RestQueryParams")
	}
}

func TestAnnotateMethod_MapPagination(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("test")
	mapEntry := api.NewTestMessage("ItemsEntry").WithFields(
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithMessageType(itemMsg),
	)
	mapField := api.NewTestField("items").WithMessageType(mapEntry).WithMap()
	nextPageToken := api.NewTestField("next_page_token").WithType(api.TypezString)
	respMsg := api.NewTestMessage("AggregatedListItemsResponse").
		WithPagination(nextPageToken, mapField)

	reqMsg := api.NewTestMessage("AggregatedListItemsRequest").WithFields(
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("page_token").WithType(api.TypezString),
	)

	method := api.NewTestMethod("AggregatedListItems").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithPagination(api.NewTestField("page_token").WithType(api.TypezString))
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, mapEntry, itemMsg}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	type paginationResult struct {
		IsPaginated     bool
		RangeOutputType string
		IsUnary         bool
	}
	want := paginationResult{
		IsPaginated:     true,
		RangeOutputType: "std::pair<std::string, test::Item>",
		IsUnary:         false,
	}
	gotResult := paginationResult{
		IsPaginated:     got.IsPaginated,
		RangeOutputType: got.RangeOutputType,
		IsUnary:         got.IsUnary,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_MapPagination_PrimitiveKey(t *testing.T) {
	itemMsg := api.NewTestMessage("Item").WithPackage("test")
	mapEntry := api.NewTestMessage("ItemsEntry").WithFields(
		api.NewTestField("key").WithType(api.TypezInt32),
		api.NewTestField("value").WithMessageType(itemMsg),
	)
	mapField := api.NewTestField("items").WithMessageType(mapEntry).WithMap()
	nextPageToken := api.NewTestField("next_page_token").WithType(api.TypezString)
	respMsg := api.NewTestMessage("AggregatedListItemsResponse").
		WithPagination(nextPageToken, mapField)

	reqMsg := api.NewTestMessage("AggregatedListItemsRequest").WithFields(
		api.NewTestField("page_size").WithType(api.TypezInt32),
		api.NewTestField("page_token").WithType(api.TypezString),
	)

	method := api.NewTestMethod("AggregatedListItems").
		WithInput(reqMsg).
		WithOutput(respMsg).
		WithPagination(api.NewTestField("page_token").WithType(api.TypezString))
	svc := api.NewTestService("ItemService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{reqMsg, respMsg, mapEntry, itemMsg}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	type paginationResult struct {
		IsPaginated     bool
		RangeOutputType string
	}
	want := paginationResult{
		IsPaginated:     true,
		RangeOutputType: "std::pair<std::int32_t, test::Item>",
	}
	gotResult := paginationResult{
		IsPaginated:     got.IsPaginated,
		RangeOutputType: got.RangeOutputType,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_ComputeLRO(t *testing.T) {
	req := api.NewTestMessage("InsertRequest")
	op := api.NewTestMessage("Operation").WithPackage("google.cloud.cpp.compute.v1")

	method := api.NewTestMethod("Insert").
		WithInput(req).
		WithOutput(op).
		WithOperationService("RegionOperations")

	svc := api.NewTestService("RegionOperationsService").WithMethods(method)
	model := api.NewTestAPI([]*api.Message{req, op}, nil, []*api.Service{svc})

	c := newCodec(nil)
	if err := c.annotateModel(model); err != nil {
		t.Fatal(err)
	}

	got := method.Codec.(*methodAnnotations)
	type computeLroResult struct {
		IsLongrunning                  bool
		IsComputeLRO                   bool
		LongrunningOperationType       string
		LongrunningDeducedResponseType string
	}
	want := computeLroResult{
		IsLongrunning:                  true,
		IsComputeLRO:                   true,
		LongrunningOperationType:       "google::cloud::cpp::compute::v1::Operation",
		LongrunningDeducedResponseType: "google::cloud::cpp::compute::v1::Operation",
	}
	gotResult := computeLroResult{
		IsLongrunning:                  got.IsLongrunning,
		IsComputeLRO:                   got.IsComputeLRO,
		LongrunningOperationType:       got.LongrunningOperationType,
		LongrunningDeducedResponseType: got.LongrunningDeducedResponseType,
	}
	if diff := cmp.Diff(want, gotResult); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
