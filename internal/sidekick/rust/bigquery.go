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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type runQueryBuilder struct {
	c      *codec
	model  *api.API
	fields []*queryField
}

type queryField struct {
	QueryRequest          *api.Field
	JobConfigurationQuery *api.Field
	JobConfiguration      *api.Field
}

func newRunQueryBuilder(c *codec, model *api.API, skippedFields []string) (*runQueryBuilder, error) {
	qrMsg := model.Message(fmt.Sprintf(".%s.QueryRequest", model.PackageName))
	jcqMsg := model.Message(fmt.Sprintf(".%s.JobConfigurationQuery", model.PackageName))
	jcMsg := model.Message(fmt.Sprintf(".%s.JobConfiguration", model.PackageName))

	if qrMsg == nil || jcqMsg == nil || jcMsg == nil {
		return nil, fmt.Errorf("failed to locate QueryRequest, JobConfigurationQuery, or JobConfiguration messages")
	}

	// Index fields by their name and collect unique names
	var allFieldNames []string
	indexFields := func(msg *api.Message) map[string]*api.Field {
		fields := make(map[string]*api.Field)
		for _, f := range msg.Fields {
			fields[f.Name] = f
			allFieldNames = append(allFieldNames, f.Name)
		}
		return fields
	}

	qrFields := indexFields(qrMsg)
	jcqFields := indexFields(jcqMsg)
	jcFields := indexFields(jcMsg)

	slices.Sort(allFieldNames)
	allFieldNames = slices.Compact(allFieldNames)

	var fields []*queryField

	for _, fieldName := range allFieldNames {
		if slices.Contains(skippedFields, fieldName) {
			continue
		}

		jcF := jcFields[fieldName]
		// Special case since JobConfigurationQuery field is also called query while it its just a string on QueryRequest
		if fieldName == "query" {
			jcF = nil
		}

		qf := &queryField{
			QueryRequest:          qrFields[fieldName],
			JobConfigurationQuery: jcqFields[fieldName],
			JobConfiguration:      jcF,
		}

		if qf.outputOnly() {
			continue
		}

		fields = append(fields, qf)
	}

	return &runQueryBuilder{
		c:      c,
		model:  model,
		fields: fields,
	}, nil
}

// createSyntheticMessage builds an api.Message populated with fields from this queryFields list
// and annotates it using the provided codec.
func (b runQueryBuilder) createSyntheticMessage(name string) (*api.Message, error) {
	msg := &api.Message{
		ID:               fmt.Sprintf(".%s.%s", b.model.PackageName, name),
		Name:             name,
		Package:          b.model.PackageName,
		SyntheticRequest: true,
	}
	for _, qf := range b.fields {
		primary := qf.firstNonNull()
		clone := *primary
		clone.Optional = qf.SourceIsOptional()
		msg.Fields = append(msg.Fields, &clone)
	}
	if err := b.c.annotateMessage(msg, b.model, true); err != nil {
		return nil, err
	}
	return msg, nil
}

func (b *runQueryBuilder) builder() (*api.Message, error) {
	msg, err := b.createSyntheticMessage("RunQuery")
	if err != nil {
		return nil, err
	}
	// TODO: check if we can avoid this hack changing data on annotations
	msgAnn, ok := msg.Codec.(*messageAnnotation)
	if !ok {
		return nil, fmt.Errorf("expected message annotation for %q", msg.ID)
	}
	for _, f := range msgAnn.BasicFields {
		fAnn, ok := f.Codec.(*fieldAnnotations)
		if !ok {
			return nil, fmt.Errorf("expected field annotation for %q", f.ID)
		}
		fAnn.FieldName = fmt.Sprintf("request.%s", fAnn.FieldName)
		fAnn.FQMessageName = "crate::model::RunQueryRequest"
	}
	return msg, nil
}

func (b *runQueryBuilder) request() (*api.Message, error) {
	return b.createSyntheticMessage("RunQueryRequest")
}

// Accessors for template files.
func (qf *queryField) JobOnly() bool {
	return qf.QueryRequest == nil
}

func (qf *queryField) Name() string {
	return qf.firstNonNull().Name
}

func (qf *queryField) Map() bool {
	return qf.firstNonNull().Map
}

func (qf *queryField) Repeated() bool {
	return qf.firstNonNull().Repeated
}

func (qf *queryField) FieldType() string {
	ann := qf.firstNonNull().Codec.(*fieldAnnotations)
	return strings.ReplaceAll(ann.FieldType, "crate::model", "google_cloud_bigquery_v2::model")
}

func (qf *queryField) SourceIsOptional() bool {
	return (qf.QueryRequest != nil && qf.QueryRequest.Optional) ||
		(qf.JobConfigurationQuery != nil && qf.JobConfigurationQuery.Optional) ||
		(qf.JobConfiguration != nil && qf.JobConfiguration.Optional)
}

func (qf *queryField) firstNonNull() *api.Field {
	for _, f := range []*api.Field{qf.QueryRequest, qf.JobConfigurationQuery, qf.JobConfiguration} {
		if f != nil {
			return f
		}
	}
	return nil
}

func (qf *queryField) outputOnly() bool {
	isOutputOnly := func(f *api.Field) bool {
		return f == nil || slices.Contains(f.Behavior, api.FieldBehaviorOutputOnly)
	}
	return isOutputOnly(qf.QueryRequest) && isOutputOnly(qf.JobConfigurationQuery) && isOutputOnly(qf.JobConfiguration)
}
