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
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestBigQueryQueryFieldOverride(t *testing.T) {
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"name-overrides": ".synthetic.google.cloud.bigquery.v2.QueryRequest.bad_query=good_query," +
			".synthetic.google.cloud.bigquery.v2.JobConfigurationQuery.bad_query=good_query," +
			".synthetic.google.cloud.bigquery.v2.JobConfiguration.bad_query=good_query," +
			".google.cloud.bigquery.v2.QueryRequest.query=gapic_query," +
			".google.cloud.bigquery.v2.JobConfigurationQuery.query=gapic_query",
	})
	if err != nil {
		t.Fatal(err)
	}

	newTestMsg := func(msgName string) *api.Message {
		queryField := api.NewTestField("query").WithType(api.TypezString)
		queryField.Codec = &fieldAnnotations{}

		overrideField := api.NewTestField("bad_query").WithType(api.TypezString)
		overrideField.Codec = &fieldAnnotations{}

		return api.NewTestMessage(msgName).
			WithPackage("google.cloud.bigquery.v2").
			WithFields(queryField, overrideField)
	}

	qrMsg := newTestMsg("QueryRequest")
	jcqMsg := newTestMsg("JobConfigurationQuery")
	jcMsg := newTestMsg("JobConfiguration")

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, nil, nil)
	builder, err := newQueryBuilder(c, model, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(builder.fieldGroups) != 2 {
		t.Fatalf("expected 2 queryFields, got %d", len(builder.fieldGroups))
	}

	fgList := builder.fieldGroupList()
	overriddenFg := fgList[0]
	if overriddenFg.FieldName() != "good_query" {
		t.Errorf("expected field name 'good_query', got %q", overriddenFg.FieldName())
	}
	if overriddenFg.GapicFieldName() != "bad_query" {
		t.Errorf("expected gapic field name 'bad_query', got %q", overriddenFg.GapicFieldName())
	}
	if !overriddenFg.QueryRequest() {
		t.Error("expected QueryRequest to be set on overriddenFg")
	}
	if !overriddenFg.JobConfigurationQuery() {
		t.Error("expected JobConfigurationQuery to be set on overriddenFg")
	}
	if !overriddenFg.JobConfiguration() {
		t.Error("expected JobConfiguration to be set on overriddenFg")
	}

	qf := fgList[1]
	if qf.FieldName() != "query" {
		t.Errorf("expected field name 'query', got %q", qf.FieldName())
	}
	if qf.GapicFieldName() != "gapic_query" {
		t.Errorf("expected gapic field name 'gapic_query', got %q", qf.GapicFieldName())
	}
	if !qf.QueryRequest() {
		t.Error("expected QueryRequest to be set")
	}
	if !qf.JobConfigurationQuery() {
		t.Error("expected JobConfigurationQuery to be set")
	}
	if qf.JobConfiguration() {
		t.Error("expected JobConfiguration to be false for field name 'query'")
	}
}

func TestBigQueryFiltering(t *testing.T) {
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	newTestField := func(name string, outputOnly bool) *api.Field {
		f := api.NewTestField(name).WithType(api.TypezString)
		if outputOnly {
			f.WithBehavior(api.FieldBehaviorOutputOnly)
		}
		f.Codec = &fieldAnnotations{}
		return f
	}
	newTestMsg := func(msgName string, fields []*api.Field) *api.Message {
		return api.NewTestMessage(msgName).
			WithPackage("google.cloud.bigquery.v2").
			WithFields(fields...)
	}

	qrMsg := newTestMsg("QueryRequest", []*api.Field{
		newTestField("output_only", true),
		newTestField("foo", false),
	})
	skipField := newTestField("skip", false)
	jcqMsg := newTestMsg("JobConfigurationQuery", []*api.Field{
		newTestField("output_only", true),
		newTestField("foo", false),
		skipField,
	})
	jcMsg := newTestMsg("JobConfiguration", []*api.Field{
		newTestField("output_only", true),
		newTestField("skip", false),
	})

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, nil, nil)
	builder, err := newQueryBuilder(c, model, []string{"skip", skipField.ID})
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
		f1 := api.NewTestField(name).WithType(api.TypezBool)
		f1.Codec = &fieldAnnotations{FieldName: name, FieldType: "bool"}
		qrFields = append(qrFields, f1)
		if i < 20 {
			f2 := api.NewTestField(name).WithType(api.TypezBool)
			f2.Codec = &fieldAnnotations{FieldName: name, FieldType: "bool"}
			jcFields = append(jcFields, f2)
		}
	}
	slices.Reverse(jcFields)

	qrMsg := api.NewTestMessage("QueryRequest").
		WithPackage("google.cloud.bigquery.v2").
		WithFields(qrFields...)
	jcqMsg := api.NewTestMessage("JobConfigurationQuery").
		WithPackage("google.cloud.bigquery.v2")
	jcMsg := api.NewTestMessage("JobConfiguration").
		WithPackage("google.cloud.bigquery.v2").
		WithFields(jcFields...)

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg}, nil, nil)
	c, err := newCodec("protobuf", map[string]string{
		"name-overrides": ".synthetic.google.cloud.bigquery.v2.QueryRequest.field_00=good_field_00," +
			".synthetic.google.cloud.bigquery.v2.JobConfiguration.field_00=good_field_00",
	})
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
	mAnn, ok := syntheticMsg.Codec.(*messageAnnotation)
	if !ok {
		t.Fatal("expected messageAnnotation on syntheticMsg")
	}
	if mAnn.HasNestedTypes {
		t.Error("expected HasNestedTypes to be false on syntheticMsg")
	}
	if syntheticMsg.Name != "MySyntheticMessage" {
		t.Errorf("expected name 'MySyntheticMessage', got %q", syntheticMsg.Name)
	}
	if len(syntheticMsg.Fields) != 40 {
		t.Fatalf("expected 40 fields, got %d", len(syntheticMsg.Fields))
	}
	for _, f := range syntheticMsg.Fields {
		wantID := fmt.Sprintf(".synthetic.google.cloud.bigquery.v2.QueryRequest.%s", f.Name)
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
	if fAnn.FieldName != "request.good_field_00" {
		t.Errorf("expected FieldName to be 'request.good_field_00', got %q", fAnn.FieldName)
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
		wantID := fmt.Sprintf(".synthetic.google.cloud.bigquery.v2.QueryRequest.%s", f.Name)
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
	if reqfAnn.FieldName != "good_field_00" {
		t.Errorf("expected FieldName to be 'good_field_00', got %q", reqfAnn.FieldName)
	}
}

func TestBigQueryOneOfs(t *testing.T) {
	c := createRustCodec()

	arrowOptsMsg := api.NewTestMessage("ArrowSerializationOptions").
		WithPackage("google.cloud.bigquery.v2")

	arrowField := api.NewTestField("arrow_serialization_options").
		WithMessageType(arrowOptsMsg)
	oneofGroup := api.NewTestOneOf("results_format_serialization_options").
		WithFields(arrowField)

	regularField := api.NewTestField("query_results_format").
		WithType(api.TypezString)

	qrMsg := api.NewTestMessage("QueryRequest").
		WithPackage("google.cloud.bigquery.v2").
		WithFields(regularField, arrowField).
		WithOneOfs(oneofGroup)

	jcqMsg := api.NewTestMessage("JobConfigurationQuery").
		WithPackage("google.cloud.bigquery.v2")
	jcMsg := api.NewTestMessage("JobConfiguration").
		WithPackage("google.cloud.bigquery.v2")

	model := api.NewTestAPI([]*api.Message{qrMsg, jcqMsg, jcMsg, arrowOptsMsg}, nil, nil)
	api.CrossReference(model)
	if _, err := annotateModel(model, c); err != nil {
		t.Fatal(err)
	}

	builder, err := newQueryBuilder(c, model, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify field groups contains oneof
	fgs := builder.fieldGroupList()
	var foundOneOf *fieldGroup
	for _, fg := range fgs {
		if fg.FieldName() == "results_format_serialization_options" {
			foundOneOf = fg
			break
		}
	}
	if foundOneOf == nil {
		t.Fatal("expected results_format_serialization_options in fieldGroupList")
	}
	if !foundOneOf.IsOneOf() {
		t.Errorf("expected foundOneOf.IsOneOf() to be true")
	}
	if !foundOneOf.QueryRequest() {
		t.Errorf("expected QueryRequest() to be true")
	}

	// Verify oneof branch fields like arrow_serialization_options are not in fieldGroupList
	for _, fg := range fgs {
		if fg.FieldName() == "arrow_serialization_options" {
			t.Errorf("unexpected branch field 'arrow_serialization_options' in fieldGroupList")
		}
	}

	// Verify synthetic QueryRequest
	queryRequest, err := builder.createSyntheticMessage("QueryRequest")
	if err != nil {
		t.Fatal(err)
	}
	if len(queryRequest.OneOfs) != 1 {
		t.Fatalf("expected 1 oneof in synthetic QueryRequest, got %d", len(queryRequest.OneOfs))
	}
	qrOneOf := queryRequest.OneOfs[0]
	qrOneOfAnn, ok := qrOneOf.Codec.(*oneOfAnnotation)
	if !ok {
		t.Fatal("expected oneOfAnnotation on synthetic QueryRequest oneof")
	}
	if qrOneOfAnn.FieldName != "results_format_serialization_options" {
		t.Errorf("expected FieldName 'results_format_serialization_options', got %q", qrOneOfAnn.FieldName)
	}
	// Verify qualified name points to GAPIC enum
	wantEnumQualifiedName := "crate::model::query_request::ResultsFormatSerializationOptions"
	if qrOneOfAnn.QualifiedName != wantEnumQualifiedName {
		t.Errorf("expected QualifiedName %q, got %q", wantEnumQualifiedName, qrOneOfAnn.QualifiedName)
	}

	// Verify synthetic Query builder message
	queryBuilderMsg, err := createQueryBuilderMessage(builder)
	if err != nil {
		t.Fatal(err)
	}
	if len(queryBuilderMsg.OneOfs) != 1 {
		t.Fatalf("expected 1 oneof in Query builder, got %d", len(queryBuilderMsg.OneOfs))
	}
	qOneOf := queryBuilderMsg.OneOfs[0]
	qOneOfAnn, ok := qOneOf.Codec.(*oneOfAnnotation)
	if !ok {
		t.Fatal("expected oneOfAnnotation on Query builder oneof")
	}
	if qOneOfAnn.FieldName != "request.results_format_serialization_options" {
		t.Errorf("expected FieldName 'request.results_format_serialization_options', got %q", qOneOfAnn.FieldName)
	}
	if qOneOfAnn.QualifiedName != wantEnumQualifiedName {
		t.Errorf("expected QualifiedName %q, got %q", wantEnumQualifiedName, qOneOfAnn.QualifiedName)
	}
}

func TestBigQueryOneOfMerging(t *testing.T) {
	c, err := newCodec("protobuf", nil)
	if err != nil {
		t.Fatal(err)
	}

	oneOfField1 := api.NewTestField("branch1").WithType(api.TypezString)
	oneOfField1.Codec = &fieldAnnotations{}
	oneOf1 := api.NewTestOneOf("my_oneof").WithFields(oneOfField1)
	oneOf1.Codec = &oneOfAnnotation{
		QualifiedName: "crate::model::MyOneOf",
	}

	oneOfField2 := api.NewTestField("branch2").WithType(api.TypezInt64)
	oneOfField2.Codec = &fieldAnnotations{}
	oneOf2 := api.NewTestOneOf("my_oneof").WithFields(oneOfField2)
	oneOf2.Codec = &oneOfAnnotation{
		QualifiedName: "crate::model::MyOneOf",
	}

	msg1 := api.NewTestMessage("Msg1").
		WithPackage("google.cloud.bigquery.v2").
		WithFields(oneOfField1).
		WithOneOfs(oneOf1)
	msg2 := api.NewTestMessage("Msg2").
		WithPackage("google.cloud.bigquery.v2").
		WithFields(oneOfField2).
		WithOneOfs(oneOf2)

	model := api.NewTestAPI([]*api.Message{msg1, msg2}, nil, nil)
	unified, err := newUnifiedMessage(c, model, []string{"Msg1", "Msg2"}, func(f *api.Field) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	if len(unified.oneofs) != 1 {
		t.Fatalf("expected 1 unified oneof, got %d", len(unified.oneofs))
	}
	mergedOneOf := unified.oneofs[0]
	if len(mergedOneOf.Fields) != 2 {
		t.Fatalf("expected 2 merged fields in unified oneof, got %d", len(mergedOneOf.Fields))
	}
	wantFields := []string{"branch1", "branch2"}
	for i, want := range wantFields {
		if mergedOneOf.Fields[i].Name != want {
			t.Errorf("field %d: expected %q, got %q", i, want, mergedOneOf.Fields[i].Name)
		}
	}
}

func TestBigQueryQueryMetadata(t *testing.T) {
	c, err := newCodec("protobuf", map[string]string{
		"name-overrides": ".synthetic.google.cloud.bigquery.v2.GetQueryResultsResponse.job_reference=job_ref_renamed," +
			".synthetic.google.cloud.bigquery.v2.Job.job_ref=job_ref_renamed",
	})
	if err != nil {
		t.Fatal(err)
	}

	newTestField := func(name string) *api.Field {
		f := api.NewTestField(name).WithType(api.TypezString)
		f.Codec = &fieldAnnotations{}
		return f
	}
	newTestMsg := func(msgName string, fields []*api.Field) *api.Message {
		return api.NewTestMessage(msgName).
			WithPackage("google.cloud.bigquery.v2").
			WithFields(fields...)
	}

	t.Run("CompleteQueryMetadata", func(t *testing.T) {
		skipByID := newTestField("skip_by_id")
		gqrMsg := newTestMsg("GetQueryResultsResponse", []*api.Field{
			newTestField("job_reference"),
			newTestField("shared_field"),
			newTestField("skip_by_name"),
			skipByID,
		})
		qrMsg := newTestMsg("QueryResponse", []*api.Field{
			newTestField("query_id"),
			newTestField("shared_field"),
		})

		model := api.NewTestAPI([]*api.Message{gqrMsg, qrMsg}, nil, nil)
		skipped := []string{"skip_by_name", skipByID.ID}

		cqm, err := newCompleteQueryMetadata(c, model, skipped)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var names []string
		var gapicNames []string
		for _, fg := range cqm.fieldGroupList() {
			names = append(names, fg.FieldName())
			gapicNames = append(gapicNames, fg.GapicFieldName())
		}
		wantNames := []string{"job_ref_renamed", "query_id", "shared_field"}
		if diff := cmp.Diff(wantNames, names); diff != "" {
			t.Errorf("field names mismatch (-want +got):\n%s", diff)
		}
		wantGapicNames := []string{"job_reference", "query_id", "shared_field"}
		if diff := cmp.Diff(wantGapicNames, gapicNames); diff != "" {
			t.Errorf("gapic field names mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("QueryMetadata", func(t *testing.T) {
		skipByID := newTestField("skip_by_id")
		jobMsg := newTestMsg("Job", []*api.Field{
			newTestField("job_ref"),
			newTestField("common_field"),
			newTestField("skip_by_name"),
			skipByID,
		})
		qrMsg := newTestMsg("QueryResponse", []*api.Field{
			newTestField("kind"),
			newTestField("common_field"),
		})

		model := api.NewTestAPI([]*api.Message{jobMsg, qrMsg}, nil, nil)
		skipped := []string{"skip_by_name", skipByID.ID}

		qm, err := newQueryMetadata(c, model, skipped)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var names []string
		var gapicNames []string
		for _, fg := range qm.fieldGroupList() {
			names = append(names, fg.FieldName())
			gapicNames = append(gapicNames, fg.GapicFieldName())
		}
		wantNames := []string{"common_field", "job_ref_renamed", "kind"}
		if diff := cmp.Diff(wantNames, names); diff != "" {
			t.Errorf("field names mismatch (-want +got):\n%s", diff)
		}
		wantGapicNames := []string{"common_field", "job_ref", "kind"}
		if diff := cmp.Diff(wantGapicNames, gapicNames); diff != "" {
			t.Errorf("gapic field names mismatch (-want +got):\n%s", diff)
		}
	})
}
