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
	oneofs      []*api.OneOf
}

type fieldGroup struct {
	gapicFieldName     string
	syntheticFieldName string
	// fields with the same name from the various messages
	fields map[string]*api.Field
	// oneofs with the same name from the various messages
	oneofs map[string]*api.OneOf
}

// newUnifiedMessage builds a new unifiedMessage from the provided message names.
// If fields have the same name across messages, the field from the message that appears
// earlier in `msgNames` is prioritized.
func newUnifiedMessage(c *codec, model *api.API, msgNames []string, skipFieldFn func(*api.Field) bool) (*unifiedMessage, error) {
	msg := &unifiedMessage{
		c:           c,
		model:       model,
		fields:      []*api.Field{},
		fieldGroups: map[string]*fieldGroup{},
		oneofs:      []*api.OneOf{},
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
			// The synthetic field can be renamed independently of the GAPIC
			// field, using a ".synthetic" prefix in the name overrides.
			gapicFieldName := c.FieldName(f)
			fClone := *f
			fClone.ID = ".synthetic" + f.ID
			syntheticFieldName := c.FieldName(&fClone)

			msg.fields = append(msg.fields, &fClone)

			// Oneof branch fields are not standalone struct fields in Rust.
			// Their enclosing oneof is added to fieldGroups below.
			if f.IsOneOf || f.Group != nil {
				continue
			}

			if _, ok := msg.fieldGroups[syntheticFieldName]; !ok {
				msg.fieldGroups[syntheticFieldName] = &fieldGroup{
					gapicFieldName:     gapicFieldName,
					syntheticFieldName: syntheticFieldName,
					fields:             make(map[string]*api.Field),
					oneofs:             make(map[string]*api.OneOf),
				}
			}
			msg.fieldGroups[syntheticFieldName].fields[msgName] = &fClone
		}

		for _, o := range m.OneOfs {
			var nonSkipped []*api.Field
			for _, f := range o.Fields {
				if skipFieldFn(f) {
					continue
				}
				nonSkipped = append(nonSkipped, f)
			}
			if len(nonSkipped) == 0 {
				continue
			}

			gapicOneOfName := c.OneOfName(o)
			oClone := *o
			oClone.ID = ".synthetic" + o.ID
			oClone.Fields = nonSkipped
			syntheticOneOfName := c.OneOfName(&oClone)

			msg.oneofs = append(msg.oneofs, &oClone)
			if _, ok := msg.fieldGroups[syntheticOneOfName]; !ok {
				msg.fieldGroups[syntheticOneOfName] = &fieldGroup{
					gapicFieldName:     gapicOneOfName,
					syntheticFieldName: syntheticOneOfName,
					fields:             make(map[string]*api.Field),
					oneofs:             make(map[string]*api.OneOf),
				}
			}
			msg.fieldGroups[syntheticOneOfName].oneofs[msgName] = o
		}
	}

	slices.SortStableFunc(msg.fields, func(a, b *api.Field) int {
		return cmp.Compare(a.Name, b.Name)
	})
	msg.fields = slices.CompactFunc(msg.fields, func(a, b *api.Field) bool {
		return a.Name == b.Name
	})

	slices.SortStableFunc(msg.oneofs, func(a, b *api.OneOf) int {
		return cmp.Compare(a.Name, b.Name)
	})
	if len(msg.oneofs) > 0 {
		compacted := []*api.OneOf{msg.oneofs[0]}
		for _, o := range msg.oneofs[1:] {
			last := compacted[len(compacted)-1]
			if last.Name == o.Name {
				for _, f := range o.Fields {
					if !slices.ContainsFunc(last.Fields, func(ef *api.Field) bool {
						return ef.Name == f.Name
					}) {
						last.Fields = append(last.Fields, f)
					}
				}
			} else {
				compacted = append(compacted, o)
			}
		}
		msg.oneofs = compacted
	}

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
	fieldsByName := make(map[string]*api.Field)
	for _, f := range m.fields {
		clone := *f
		msg.Fields = append(msg.Fields, &clone)
		fieldsByName[clone.Name] = &clone
	}
	for _, o := range m.oneofs {
		oClone := *o
		oClone.Fields = nil
		for _, f := range o.Fields {
			if fMatch, ok := fieldsByName[f.Name]; ok {
				fMatch.Group = &oClone
				fMatch.IsOneOf = true
				oClone.Fields = append(oClone.Fields, fMatch)
			}
		}
		if len(oClone.Fields) > 0 {
			msg.OneOfs = append(msg.OneOfs, &oClone)
		}
	}
	if err := m.c.annotateMessage(msg, m.model, true); err != nil {
		return nil, err
	}
	if mAnn, ok := msg.Codec.(*messageAnnotation); ok {
		mAnn.HasNestedTypes = false
	}
	// synthetic messages are not in the model, so we need to set their FQMessageName
	// to point to crate::model_ext::{name}
	for _, f := range msg.Fields {
		fAnn, ok := f.Codec.(*fieldAnnotations)
		if !ok {
			return nil, fmt.Errorf("expected field annotation for %q", f.ID)
		}
		fAnn.FQMessageName = fmt.Sprintf("crate::model_ext::%s", name)
		fAnn.GenerateSetterSamples = false
	}
	for _, o := range msg.OneOfs {
		oAnn, ok := o.Codec.(*oneOfAnnotation)
		if !ok {
			return nil, fmt.Errorf("expected oneof annotation for %q", o.ID)
		}
		oneofName := m.c.OneOfName(o)
		if g, ok := m.fieldGroups[oneofName]; ok {
			for _, origOneOf := range g.oneofs {
				if origAnn, ok := origOneOf.Codec.(*oneOfAnnotation); ok {
					oAnn.QualifiedName = origAnn.QualifiedName
					oAnn.FieldType = origAnn.FieldType
					oAnn.EnumName = origAnn.EnumName
					oAnn.RelativeName = origAnn.RelativeName
					oAnn.NameInExamples = origAnn.NameInExamples
					oAnn.EnumNameInExamples = origAnn.EnumNameInExamples
					break
				}
			}
		}
		oAnn.StructQualifiedName = fmt.Sprintf("crate::model_ext::%s", name)
		oAnn.GenerateSetterSamples = false
	}
	return msg, nil
}

func (m *unifiedMessage) fieldGroupList() []*fieldGroup {
	list := make([]*fieldGroup, 0, len(m.fieldGroups))
	for _, g := range m.fieldGroups {
		list = append(list, g)
	}
	slices.SortStableFunc(list, func(a, b *fieldGroup) int {
		return cmp.Compare(a.FieldName(), b.FieldName())
	})
	return list
}

func newQueryBuilder(c *codec, model *api.API, skippedFields []string) (*unifiedMessage, error) {
	msg, err := newUnifiedMessage(c, model, []string{"QueryRequest", "JobConfigurationQuery", "JobConfiguration"}, func(f *api.Field) bool {
		// skip fields that are output only or explicitly skipped
		return slices.Contains(skippedFields, f.Name) || slices.Contains(skippedFields, f.ID) || slices.Contains(f.Behavior, api.FieldBehaviorOutputOnly)
	})
	if err != nil {
		return nil, err
	}

	// The `query` field is a string on `QueryRequest`, but a `JobConfigurationQuery` message on `JobConfiguration`
	if f, ok := msg.fieldGroups["query"]; ok {
		delete(f.fields, "JobConfiguration")
	}

	return msg, nil
}

func newCompleteQueryMetadata(c *codec, model *api.API, skippedFields []string) (*unifiedMessage, error) {
	return newUnifiedMessage(c, model, []string{"GetQueryResultsResponse", "QueryResponse"}, func(f *api.Field) bool {
		return slices.Contains(skippedFields, f.Name) || slices.Contains(skippedFields, f.ID)
	})
}

func newQueryMetadata(c *codec, model *api.API, skippedFields []string) (*unifiedMessage, error) {
	return newUnifiedMessage(c, model, []string{"Job", "QueryResponse"}, func(f *api.Field) bool {
		return slices.Contains(skippedFields, f.Name) || slices.Contains(skippedFields, f.ID)
	})
}

func createQueryBuilderMessage(m *unifiedMessage) (*api.Message, error) {
	msg, err := m.createSyntheticMessage("Query")
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
		fAnn.FQMessageName = "crate::builder::bigquery::QueryRequest"
	}
	for _, o := range msg.OneOfs {
		oAnn, ok := o.Codec.(*oneOfAnnotation)
		if !ok {
			return nil, fmt.Errorf("expected oneof annotation for %q", o.ID)
		}
		oAnn.FieldName = fmt.Sprintf("request.%s", oAnn.FieldName)
		oAnn.StructQualifiedName = "crate::builder::bigquery::QueryRequest"
		for _, f := range o.Fields {
			fAnn, ok := f.Codec.(*fieldAnnotations)
			if !ok {
				return nil, fmt.Errorf("expected field annotation for %q", f.ID)
			}
			fAnn.FQMessageName = "crate::builder::bigquery::QueryRequest"
		}
	}
	return msg, nil
}

// Accessors for template files.
func (f *fieldGroup) FieldName() string {
	return toSnake(f.syntheticFieldName)
}

func (f *fieldGroup) GapicFieldName() string {
	return toSnake(f.gapicFieldName)
}

func (f *fieldGroup) IsOneOf() bool {
	return len(f.oneofs) > 0
}

func (f *fieldGroup) JobOnly() bool {
	return !f.QueryRequest()
}

func (f *fieldGroup) JobConfiguration() bool {
	return f.fields["JobConfiguration"] != nil || f.oneofs["JobConfiguration"] != nil
}

func (f *fieldGroup) QueryRequest() bool {
	return f.fields["QueryRequest"] != nil || f.oneofs["QueryRequest"] != nil
}

func (f *fieldGroup) JobConfigurationQuery() bool {
	return f.fields["JobConfigurationQuery"] != nil || f.oneofs["JobConfigurationQuery"] != nil
}

func (f *fieldGroup) Job() bool {
	return f.fields["Job"] != nil || f.oneofs["Job"] != nil
}

func (f *fieldGroup) QueryResponse() bool {
	return f.fields["QueryResponse"] != nil || f.oneofs["QueryResponse"] != nil
}

func (f *fieldGroup) GetQueryResultsResponse() bool {
	return f.fields["GetQueryResultsResponse"] != nil || f.oneofs["GetQueryResultsResponse"] != nil
}
