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
	"cmp"
	"fmt"
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type unifiedMessage struct {
	c           *codec
	model       *api.API
	fields      []*api.Field
	fieldGroups map[string]*fieldGroup
}

type fieldGroup struct {
	fieldName string
	// fields with the same name from the various messages
	fields map[string]*api.Field
}

func newUnifiedMessage(c *codec, model *api.API, msgNames []string, skipFieldFn func(*api.Field) bool) (*unifiedMessage, error) {
	msg := &unifiedMessage{
		c:           c,
		model:       model,
		fields:      []*api.Field{},
		fieldGroups: map[string]*fieldGroup{},
	}

	// Index fields by their name and collect names
	for _, msgName := range msgNames {
		m := model.Message(fmt.Sprintf(".%s.%s", model.PackageName, msgName))
		if m == nil {
			return nil, fmt.Errorf("failed to locate message %q", msgName)
		}
		for _, f := range m.Fields {
			if skipFieldFn(f) {
				continue
			}
			msg.fields = append(msg.fields, f)
			if _, ok := msg.fieldGroups[f.Name]; !ok {
				msg.fieldGroups[f.Name] = &fieldGroup{
					fieldName: f.Name,
					fields:    make(map[string]*api.Field),
				}
			}
			msg.fieldGroups[f.Name].fields[msgName] = f
		}
	}

	slices.SortFunc(msg.fields, func(a, b *api.Field) int {
		return cmp.Compare(a.Name, b.Name)
	})
	msg.fields = slices.CompactFunc(msg.fields, func(a, b *api.Field) bool {
		return a.Name == b.Name
	})

	return msg, nil
}

// createSyntheticMessage builds an api.Message populated with all sorted fields
// and annotates it using the provided codec.
func (m *unifiedMessage) createSyntheticMessage(name string) (*api.Message, error) {
	msg := &api.Message{
		ID:               fmt.Sprintf(".%s.%s", m.model.PackageName, name),
		Name:             name,
		Package:          m.model.PackageName,
		SyntheticRequest: true,
	}
	for _, f := range m.fields {
		clone := *f
		msg.Fields = append(msg.Fields, &clone)
	}
	if err := m.c.annotateMessage(msg, m.model, true); err != nil {
		return nil, err
	}
	return msg, nil
}

func (m *unifiedMessage) fieldGroupList() []*fieldGroup {
	list := make([]*fieldGroup, 0, len(m.fields))
	for _, f := range m.fields {
		list = append(list, m.fieldGroups[f.Name])
	}
	return list
}

func newRunQuery(c *codec, model *api.API, skippedFields []string) (*unifiedMessage, error) {
	msg, err := newUnifiedMessage(c, model, []string{"QueryRequest", "JobConfigurationQuery", "JobConfiguration"}, func(f *api.Field) bool {
		// skip fields that are output only or explicitly skipped
		return slices.Contains(skippedFields, f.Name) || slices.Contains(f.Behavior, api.FieldBehaviorOutputOnly)
	})
	if err != nil {
		return nil, err
	}

	// Special case since JobConfigurationQuery field is also called query while it its just a string on QueryRequest
	if f, ok := msg.fieldGroups["query"]; ok {
		delete(f.fields, "JobConfiguration")
	}

	return msg, nil
}

func newQueryMetadata(c *codec, model *api.API, skippedFields []string) (*unifiedMessage, error) {
	return newUnifiedMessage(c, model, []string{"GetQueryResultsResponse", "QueryResponse"}, func(f *api.Field) bool {
		return slices.Contains(skippedFields, f.Name)
	})
}

func runQueryBuilder(m *unifiedMessage) (*api.Message, error) {
	msg, err := m.createSyntheticMessage("RunQuery")
	if err != nil {
		return nil, err
	}
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

// Accessors for template files.
func (f *fieldGroup) FieldName() string {
	return toSnake(f.fieldName)
}

func (f *fieldGroup) JobOnly() bool {
	return f.QueryRequest() == nil
}

func (f *fieldGroup) JobConfiguration() *api.Field {
	return f.fields["JobConfiguration"]
}

func (f *fieldGroup) QueryRequest() *api.Field {
	return f.fields["QueryRequest"]
}

func (f *fieldGroup) JobConfigurationQuery() *api.Field {
	return f.fields["JobConfigurationQuery"]
}

func (f *fieldGroup) QueryResponse() *api.Field {
	return f.fields["QueryResponse"]
}

func (f *fieldGroup) GetQueryResultsResponse() *api.Field {
	return f.fields["GetQueryResultsResponse"]
}
