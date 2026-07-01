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

package rust

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestBigQueryQueryFieldOverride(t *testing.T) {
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	newTestMsgWithQuery := func(msgName string) *api.Message {
		field := &api.Field{Name: "query", Codec: &fieldAnnotations{}}

		return &api.Message{
			ID:      ".google.cloud.bigquery.v2." + msgName,
			Name:    msgName,
			Package: "google.cloud.bigquery.v2",
			Fields:  []*api.Field{field},
		}
	}

	qrMsg := newTestMsgWithQuery("QueryRequest")
	jcqMsg := newTestMsgWithQuery("JobConfigurationQuery")
	jcMsg := newTestMsgWithQuery("JobConfiguration")

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, []*api.Enum{}, []*api.Service{})
	builder, err := newRunQuery(c, model, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(builder.fieldGroups) != 1 {
		t.Fatalf("expected 1 queryField, got %d", len(builder.fieldGroups))
	}

	qf := builder.fieldGroupList()[0]
	if qf.FieldName() != "query" {
		t.Errorf("expected field name 'query', got %q", qf.FieldName())
	}
	if qf.QueryRequest() == nil {
		t.Error("expected QueryRequest to be set")
	}
	if qf.JobConfigurationQuery() == nil {
		t.Error("expected JobConfigurationQuery to be set")
	}
	if qf.JobConfiguration() != nil {
		t.Error("expected JobConfiguration to be nil for field name 'query'")
	}
}

func TestBigQueryFiltering(t *testing.T) {
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	newTestField := func(name string, outputOnly bool) *api.Field {
		b := []api.FieldBehavior{}
		if outputOnly {
			b = append(b, api.FieldBehaviorOutputOnly)
		}
		return &api.Field{
			Name:     name,
			Behavior: b,
			Codec:    &fieldAnnotations{},
		}
	}
	newTestMsg := func(msgName string, fields []*api.Field) *api.Message {
		return &api.Message{
			ID:      ".google.cloud.bigquery.v2." + msgName,
			Name:    msgName,
			Package: "google.cloud.bigquery.v2",
			Fields:  fields,
		}
	}

	qrMsg := newTestMsg("QueryRequest", []*api.Field{newTestField("output_only", true), newTestField("foo", false)})
	jcqMsg := newTestMsg("JobConfigurationQuery", []*api.Field{newTestField("output_only", true), newTestField("foo", false)})
	jcMsg := newTestMsg("JobConfiguration", []*api.Field{newTestField("output_only", true), newTestField("skip", false)})

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, []*api.Enum{}, []*api.Service{})
	builder, err := newRunQuery(c, model, []string{"skip"})
	if err != nil {
		t.Fatal(err)
	}

	var fieldNames []string
	for _, f := range builder.fieldGroupList() {
		fieldNames = append(fieldNames, f.FieldName())
	}

	// "output_only" and "skip" must be skipped; "foo" must be present.
	want := []string{"foo"}
	if diff := cmp.Diff(want, fieldNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBigQuerySyntheticMessages(t *testing.T) {
	f1QR := &api.Field{
		ID:    ".google.cloud.bigquery.v2.QueryRequest.foo",
		Name:  "foo",
		Typez: api.TypezBool,
		Codec: &fieldAnnotations{FieldName: "foo", FieldType: "bool"},
	}

	qrMsg := &api.Message{
		ID:      ".google.cloud.bigquery.v2.QueryRequest",
		Name:    "QueryRequest",
		Package: "google.cloud.bigquery.v2",
		Fields:  []*api.Field{f1QR},
	}
	jcqMsg := &api.Message{
		ID:      ".google.cloud.bigquery.v2.JobConfigurationQuery",
		Name:    "JobConfigurationQuery",
		Package: "google.cloud.bigquery.v2",
		Fields:  []*api.Field{},
	}
	jcMsg := &api.Message{
		ID:      ".google.cloud.bigquery.v2.JobConfiguration",
		Name:    "JobConfiguration",
		Package: "google.cloud.bigquery.v2",
		Fields:  []*api.Field{},
	}

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, []*api.Enum{}, []*api.Service{})
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	builder, err := newRunQuery(c, model, nil)
	if err != nil {
		t.Fatal(err)
	}

	syntheticMsg, err := builder.createSyntheticMessage("MySyntheticMessage")
	if err != nil {
		t.Fatal(err)
	}
	if !syntheticMsg.SyntheticRequest {
		t.Error("expected SyntheticRequest to be true")
	}
	if syntheticMsg.Name != "MySyntheticMessage" {
		t.Errorf("expected name 'MySyntheticMessage', got %q", syntheticMsg.Name)
	}
	if len(syntheticMsg.Fields) != 1 || syntheticMsg.Fields[0].Name != "foo" {
		t.Fatalf("expected 1 field named 'foo'")
	}

	// 2. Verify builder() output has modified basic field annotations
	runQuery, err := runQueryBuilder(builder)
	if err != nil {
		t.Fatal(err)
	}
	if runQuery.Name != "RunQuery" {
		t.Errorf("expected name 'RunQuery', got %q", runQuery.Name)
	}
	msgAnn, ok := runQuery.Codec.(*messageAnnotation)
	if !ok {
		t.Fatalf("expected messageAnnotation on RunQuery msg")
	}
	if len(msgAnn.BasicFields) != 1 {
		t.Fatalf("expected 1 basic field annotation, got %d", len(msgAnn.BasicFields))
	}
	fAnn, ok := msgAnn.BasicFields[0].Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("expected fieldAnnotations on the basic field")
	}
	if fAnn.FieldName != "request.foo" {
		t.Errorf("expected FieldName to be 'request.foo', got %q", fAnn.FieldName)
	}
	if fAnn.FQMessageName != "crate::model::RunQueryRequest" {
		t.Errorf("expected FQMessageName to be 'crate::model::RunQueryRequest', got %q", fAnn.FQMessageName)
	}

	runQueryRequest, err := builder.createSyntheticMessage("RunQueryRequest")
	if err != nil {
		t.Fatal(err)
	}
	if runQueryRequest.Name != "RunQueryRequest" {
		t.Errorf("expected name 'RunQueryRequest', got %q", runQueryRequest.Name)
	}
	reqMsgAnn, ok := runQueryRequest.Codec.(*messageAnnotation)
	if !ok {
		t.Fatalf("expected messageAnnotation on RunQueryRequest msg")
	}
	if len(reqMsgAnn.BasicFields) != 1 {
		t.Fatalf("expected 1 basic field annotation, got %d", len(reqMsgAnn.BasicFields))
	}
	reqfAnn, ok := reqMsgAnn.BasicFields[0].Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("expected fieldAnnotations on the basic field")
	}
	// Annotations should remain unchanged (not prepended with 'request.')
	if reqfAnn.FieldName != "foo" {
		t.Errorf("expected FieldName to remain 'foo', got %q", reqfAnn.FieldName)
	}
}
