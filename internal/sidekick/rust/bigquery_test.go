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
	"fmt"
	"slices"
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
	builder, err := newQueryBuilder(c, model, nil)
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

	newTestField := func(name, id string, outputOnly bool) *api.Field {
		b := []api.FieldBehavior{}
		if outputOnly {
			b = append(b, api.FieldBehaviorOutputOnly)
		}
		return &api.Field{
			ID:       id,
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

	qrMsg := newTestMsg("QueryRequest", []*api.Field{
		newTestField("output_only", ".google.cloud.bigquery.v2.QueryRequest.output_only", true),
		newTestField("foo", ".google.cloud.bigquery.v2.QueryRequest.foo", false),
	})
	jcqMsg := newTestMsg("JobConfigurationQuery", []*api.Field{
		newTestField("output_only", ".google.cloud.bigquery.v2.JobConfigurationQuery.output_only", true),
		newTestField("foo", ".google.cloud.bigquery.v2.JobConfigurationQuery.foo", false),
		newTestField("skip", ".google.cloud.bigquery.v2.JobConfigurationQuery.skip", false),
	})
	jcMsg := newTestMsg("JobConfiguration", []*api.Field{
		newTestField("output_only", ".google.cloud.bigquery.v2.JobConfiguration.output_only", true),
		newTestField("skip", ".google.cloud.bigquery.v2.JobConfiguration.skip", false),
	})

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, []*api.Enum{}, []*api.Service{})
	builder, err := newQueryBuilder(c, model, []string{"skip", ".google.cloud.bigquery.v2.JobConfigurationQuery.skip"})
	if err != nil {
		t.Fatal(err)
	}

	var fieldNames []string
	for _, f := range builder.fieldGroupList() {
		fieldNames = append(fieldNames, f.FieldName())
	}

	// "output_only", "skip" from JobConfiguration (by name) and "skip" from
	// JobConfigurationQuery (by ID) must be filtered out.
	// "foo" must be present.
	want := []string{"foo"}
	if diff := cmp.Diff(want, fieldNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestBigQuerySyntheticMessages(t *testing.T) {
	var qrFields []*api.Field
	var jcFields []*api.Field
	// make QueryRequest have 40 fields and JobConfiguration have 20 fields.
	// this causes stable sort order to matter and de-duplication to be exercised.
	for i := range 40 {
		name := fmt.Sprintf("field_%02d", i)
		qrFields = append(qrFields, &api.Field{
			ID:    fmt.Sprintf(".google.cloud.bigquery.v2.QueryRequest.%s", name),
			Name:  name,
			Typez: api.TypezBool,
			Codec: &fieldAnnotations{FieldName: name, FieldType: "bool"},
		})
		if i < 20 {
			jcFields = append(jcFields, &api.Field{
				ID:    fmt.Sprintf(".google.cloud.bigquery.v2.JobConfiguration.%s", name),
				Name:  name,
				Typez: api.TypezBool,
				Codec: &fieldAnnotations{FieldName: name, FieldType: "bool"},
			})
		}
	}
	slices.Reverse(jcFields)

	qrMsg := &api.Message{
		ID:      ".google.cloud.bigquery.v2.QueryRequest",
		Name:    "QueryRequest",
		Package: "google.cloud.bigquery.v2",
		Fields:  qrFields,
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
		Fields:  jcFields,
	}

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, []*api.Enum{}, []*api.Service{})
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	builder, err := newQueryBuilder(c, model, nil)
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
	if len(syntheticMsg.Fields) != 40 {
		t.Fatalf("expected 40 fields, got %d", len(syntheticMsg.Fields))
	}
	for _, f := range syntheticMsg.Fields {
		wantID := fmt.Sprintf(".google.cloud.bigquery.v2.QueryRequest.%s", f.Name)
		if f.ID != wantID {
			t.Errorf("expected field ID %q, got %q", wantID, f.ID)
		}
	}

	// Verify that synthetic messages point to crate::model_ext
	for _, f := range syntheticMsg.Fields {
		fAnn, ok := f.Codec.(*fieldAnnotations)
		if !ok {
			t.Fatalf("expected fieldAnnotations on the field %q", f.ID)
		}
		if fAnn.FQMessageName != "crate::model_ext::MySyntheticMessage" {
			t.Errorf("expected FQMessageName to be 'crate::model_ext::MySyntheticMessage', got %q", fAnn.FQMessageName)
		}
	}

	// Verify builder() output has modified basic field annotations
	queryBuilder, err := createQueryBuilderMessage(builder)
	if err != nil {
		t.Fatal(err)
	}
	if queryBuilder.Name != "Query" {
		t.Errorf("expected name 'Query', got %q", queryBuilder.Name)
	}
	msgAnn, ok := queryBuilder.Codec.(*messageAnnotation)
	if !ok {
		t.Fatalf("expected messageAnnotation on Query msg")
	}
	if len(msgAnn.BasicFields) != 40 {
		t.Fatalf("expected 40 basic field annotations, got %d", len(msgAnn.BasicFields))
	}
	fAnn, ok := msgAnn.BasicFields[0].Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("expected fieldAnnotations on the basic field")
	}
	if fAnn.FieldName != "request.field_00" {
		t.Errorf("expected FieldName to be 'request.field_00', got %q", fAnn.FieldName)
	}
	if fAnn.FQMessageName != "crate::builder::bigquery::QueryRequest" {
		t.Errorf("expected FQMessageName to be 'crate::builder::bigquery::QueryRequest', got %q", fAnn.FQMessageName)
	}

	queryRequest, err := builder.createSyntheticMessage("QueryRequest")
	if err != nil {
		t.Fatal(err)
	}
	if queryRequest.Name != "QueryRequest" {
		t.Errorf("expected name 'QueryRequest', got %q", queryRequest.Name)
	}
	for _, f := range queryRequest.Fields {
		wantID := fmt.Sprintf(".google.cloud.bigquery.v2.QueryRequest.%s", f.Name)
		if f.ID != wantID {
			t.Errorf("expected field ID %q, got %q", wantID, f.ID)
		}
	}
	reqMsgAnn, ok := queryRequest.Codec.(*messageAnnotation)
	if !ok {
		t.Fatalf("expected messageAnnotation on RunQueryRequest msg")
	}
	if len(reqMsgAnn.BasicFields) != 40 {
		t.Fatalf("expected 40 basic field annotations, got %d", len(reqMsgAnn.BasicFields))
	}
	reqfAnn, ok := reqMsgAnn.BasicFields[0].Codec.(*fieldAnnotations)
	if !ok {
		t.Fatalf("expected fieldAnnotations on the basic field")
	}
	if reqfAnn.FieldName != "field_00" {
		t.Errorf("expected FieldName to remain 'field_00', got %q", reqfAnn.FieldName)
	}
}

func TestBigQueryQueryMetadata(t *testing.T) {
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	newTestField := func(name, id string) *api.Field {
		return &api.Field{
			ID:    id,
			Name:  name,
			Codec: &fieldAnnotations{},
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

	t.Run("CompleteQueryMetadata", func(t *testing.T) {
		gqrMsg := newTestMsg("GetQueryResultsResponse", []*api.Field{
			newTestField("job_reference", ".google.cloud.bigquery.v2.GetQueryResultsResponse.job_reference"),
			newTestField("shared_field", ".google.cloud.bigquery.v2.GetQueryResultsResponse.shared_field"),
			newTestField("skip_by_name", ".google.cloud.bigquery.v2.GetQueryResultsResponse.skip_by_name"),
			newTestField("skip_by_id", ".google.cloud.bigquery.v2.GetQueryResultsResponse.skip_by_id"),
		})
		qrMsg := newTestMsg("QueryResponse", []*api.Field{
			newTestField("query_id", ".google.cloud.bigquery.v2.QueryResponse.query_id"),
			newTestField("shared_field", ".google.cloud.bigquery.v2.QueryResponse.shared_field"),
		})

		model := api.NewTestAPI([]*api.Message{gqrMsg, qrMsg}, []*api.Enum{}, []*api.Service{})
		skipped := []string{"skip_by_name", ".google.cloud.bigquery.v2.GetQueryResultsResponse.skip_by_id"}

		cqm, err := newCompleteQueryMetadata(c, model, skipped)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var names []string
		for _, fg := range cqm.fieldGroupList() {
			names = append(names, fg.name)
		}
		wantNames := []string{"job_reference", "query_id", "shared_field"}
		if diff := cmp.Diff(wantNames, names); diff != "" {
			t.Errorf("field names mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("QueryMetadata", func(t *testing.T) {
		jobMsg := newTestMsg("Job", []*api.Field{
			newTestField("job_ref", ".google.cloud.bigquery.v2.Job.job_ref"),
			newTestField("common_field", ".google.cloud.bigquery.v2.Job.common_field"),
			newTestField("skip_by_name", ".google.cloud.bigquery.v2.Job.skip_by_name"),
			newTestField("skip_by_id", ".google.cloud.bigquery.v2.Job.skip_by_id"),
		})
		qrMsg := newTestMsg("QueryResponse", []*api.Field{
			newTestField("kind", ".google.cloud.bigquery.v2.QueryResponse.kind"),
			newTestField("common_field", ".google.cloud.bigquery.v2.QueryResponse.common_field"),
		})

		model := api.NewTestAPI([]*api.Message{jobMsg, qrMsg}, []*api.Enum{}, []*api.Service{})
		skipped := []string{"skip_by_name", ".google.cloud.bigquery.v2.Job.skip_by_id"}

		qm, err := newQueryMetadata(c, model, skipped)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var names []string
		for _, fg := range qm.fieldGroupList() {
			names = append(names, fg.name)
		}
		wantNames := []string{"common_field", "job_ref", "kind"}
		if diff := cmp.Diff(wantNames, names); diff != "" {
			t.Errorf("field names mismatch (-want +got):\n%s", diff)
		}
	})
}
