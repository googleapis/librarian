// Copyright 2025 Google LLC
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

package api

import (
	"fmt"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
)

// CrossReference fills out the cross-references in `model` that the parser(s)
// missed.
//
// The parsers cannot always cross-reference all elements because the
// elements are built incrementally, and may not be available until the parser
// has completed all the work.
//
// This function is called after the parser has completed its work but before
// the codecs run. It populates links between the parsed elements that the
// codecs need. For example, the `oneof` fields use the containing `OneOf` to
// reference any types or names of the `OneOf` during their generation.
func CrossReference(model *API) error {
	for _, m := range model.State.MessageByID {
		for _, f := range m.Fields {
			f.Parent = m
			switch f.Typez {
			case MESSAGE_TYPE:
				t, ok := model.State.MessageByID[f.TypezID]
				if !ok {
					return fmt.Errorf("cannot find message type %s for field %s", f.TypezID, f.ID)
				}
				f.MessageType = t
			case ENUM_TYPE:
				t, ok := model.State.EnumByID[f.TypezID]
				if !ok {
					return fmt.Errorf("cannot find enum type %s for field %s", f.TypezID, f.ID)
				}
				f.EnumType = t
			}
		}
		for _, o := range m.OneOfs {
			for _, f := range o.Fields {
				f.Group = o
				f.Parent = m
			}
		}
	}
	for _, m := range model.State.MethodByID {
		input, ok := model.State.MessageByID[m.InputTypeID]
		if !ok {
			return fmt.Errorf("cannot find input type %s for method %s", m.InputTypeID, m.ID)
		}
		output, ok := model.State.MessageByID[m.OutputTypeID]
		if !ok {
			return fmt.Errorf("cannot find output type %s for method %s", m.OutputTypeID, m.ID)
		}
		m.InputType = input
		m.OutputType = output
		if m.OperationInfo != nil {
			m.OperationInfo.Method = m
		}
	}
	for _, s := range model.State.ServiceByID {
		s.Model = model
		for _, m := range s.Methods {
			m.Model = model
			m.Service = s
			source, ok := model.State.ServiceByID[m.SourceServiceID]
			if ok {
				m.SourceService = source
			} else {
				// Default to the regular service. OpenAPI does not define the
				// services for mixins.
				m.SourceService = s
			}
		}
	}
	enrichSamples(model)
	return nil
}

// enrichSamples populates the API model with information useful for generating code samples.
// This includes selecting representative enum values and optimal fields for oneof structures.
func enrichSamples(model *API) {
	for _, e := range model.State.EnumByID {
		enrichEnumSamples(e)
	}

	for _, m := range model.State.MessageByID {
		for _, o := range m.OneOfs {
			if len(o.Fields) > 0 {
				o.ExampleField = slices.MaxFunc(o.Fields, sortOneOfFieldForExamples)
			}
		}
	}

	for _, m := range model.State.MethodByID {
		enrichMethodSamples(m)
	}
}

func enrichEnumSamples(e *Enum) {
	// We try to pick some good enum values to show in examples.
	// - We pick values that are not deprecated.
	// - We don't pick the default value (Number 0).
	// - We try to avoid duplicates (e.g. FULL vs full).

	// First, deduplicate by normalized name, keeping the "best" version.
	// We prefer values that are not deprecated and not zero.
	bestByNorm := make(map[string]*EnumValue)
	var orderedNorms []string

	isGood := func(v *EnumValue) bool {
		return !v.Deprecated && v.Number != 0
	}

	for _, ev := range e.Values {
		// A simple heuristic to avoid duplicates.
		// This is not perfect, but it should handle the most common cases.
		name := strcase.ToCamel(strings.ToLower(ev.Name))
		existing, ok := bestByNorm[name]
		if !ok {
			bestByNorm[name] = ev
			orderedNorms = append(orderedNorms, name)
			continue
		}
		// If the existing one is "bad" and the new one is "good", replace it.
		// If both are good or both are bad, we keep the first one (existing).
		if isGood(ev) && !isGood(existing) {
			bestByNorm[name] = ev
		}
	}

	var goodValues []*EnumValue
	var badValues []*EnumValue

	for _, name := range orderedNorms {
		ev := bestByNorm[name]
		if isGood(ev) {
			goodValues = append(goodValues, ev)
		} else {
			badValues = append(badValues, ev)
		}
	}

	// Combine: prefer good values.
	// If we found any good values, use them. Otherwise, use the bad values (fallback).
	result := goodValues
	if len(result) == 0 {
		result = badValues
	}

	// We pick at most 3 values as samples do not need to be exhaustive.
	if len(result) > 3 {
		result = result[:3]
	}

	e.ValuesForExamples = make([]*SampleValue, len(result))
	for i, ev := range result {
		e.ValuesForExamples[i] = &SampleValue{
			EnumValue: ev,
			Index:     i,
		}
	}
}

// sortOneOfFieldForExamples is used to select the "best" field for an example.
//
// Fields are lexicographically sorted by the tuple:
//
//	(f.Deprecated, f.Map, f.Repeated, f.Message != nil)
//
// Where `false` values are preferred over `true` values. That is, we prefer
// fields that are **not** deprecated, but if both fields have the same
// `Deprecated` value then we prefer the field that is **not** a map, and so on.
//
// The return value is either -1, 0, or 1 to use in the standard library sorting
// functions.
func sortOneOfFieldForExamples(f1, f2 *Field) int {
	compare := func(a, b bool) int {
		switch {
		case a == b:
			return 0
		case a:
			return -1
		default:
			return 1
		}
	}
	if v := compare(f1.Deprecated, f2.Deprecated); v != 0 {
		return v
	}
	if v := compare(f1.Map, f2.Map); v != 0 {
		return v
	}
	if v := compare(f1.Repeated, f2.Repeated); v != 0 {
		return v
	}
	return compare(f1.MessageType != nil, f2.MessageType != nil)
}

func enrichMethodSamples(m *Method) {
	m.IsSimple = m.Pagination == nil &&
		!m.ClientSideStreaming && !m.ServerSideStreaming &&
		m.OperationInfo == nil && m.DiscoveryLro == nil

	m.IsLRO = m.OperationInfo != nil

	if m.OperationInfo != nil && m.Model != nil && m.Model.State != nil {
		m.LongRunningResponseType = m.Model.State.MessageByID[m.OperationInfo.ResponseTypeID]
	}

	m.LongRunningReturnsEmpty = m.LongRunningResponseType != nil && m.LongRunningResponseType.ID == ".google.protobuf.Empty"

	m.IsList = m.OutputType != nil && m.OutputType.Pagination != nil

	m.IsStreaming = m.ClientSideStreaming || m.ServerSideStreaming

	m.AIPStandardGetInfo = aipStandardGetInfo(m)
	m.AIPStandardDeleteInfo = aipStandardDeleteInfo(m)
	m.AIPStandardUndeleteInfo = aipStandardUndeleteInfo(m)
	m.AIPStandardCreateInfo = aipStandardCreateInfo(m)
	m.AIPStandardUpdateInfo = aipStandardUpdateInfo(m)
	m.AIPStandardListInfo = aipStandardListInfo(m)

	m.IsAIPStandard = m.AIPStandardGetInfo != nil ||
		m.AIPStandardDeleteInfo != nil ||
		m.AIPStandardUndeleteInfo != nil ||
		m.AIPStandardListInfo != nil ||
		m.AIPStandardCreateInfo != nil ||
		m.AIPStandardUpdateInfo != nil
}

func aipStandardGetInfo(m *Method) *AIPStandardGetInfo {
	if !m.IsSimple || m.InputType == nil || m.ReturnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "get")
	if !found || maybeSingular == "" {
		return nil
	}
	if strings.ToLower(m.InputType.Name) != fmt.Sprintf("get%srequest", maybeSingular) {
		return nil
	}

	if outputResource.Singular != "" &&
		strings.ToLower(outputResource.Singular) != maybeSingular {
		return nil
	}

	var resourceByType map[string]*Resource
	if m.Model != nil && m.Model.State != nil {
		resourceByType = m.Model.State.ResourceByType
	}

	resourceField := findBestResourceFieldByType(m.InputType, resourceByType, outputResource.Type)

	if resourceField == nil {
		return nil
	}

	return &AIPStandardGetInfo{
		ResourceNameRequestField: resourceField,
	}
}

func aipStandardDeleteInfo(m *Method) *AIPStandardDeleteInfo {
	if !m.IsSimple && m.OperationInfo == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "delete")
	if !found || maybeSingular == "" {
		return nil
	}
	if m.InputType == nil ||
		strings.ToLower(m.InputType.Name) != fmt.Sprintf("delete%srequest", maybeSingular) {
		return nil
	}

	var resourceByType map[string]*Resource
	if m.Model != nil && m.Model.State != nil {
		resourceByType = m.Model.State.ResourceByType
	}

	resourceField := findBestResourceFieldBySingular(m.InputType, resourceByType, maybeSingular)
	if resourceField == nil {
		return nil
	}

	return &AIPStandardDeleteInfo{
		ResourceNameRequestField: resourceField,
	}
}

func aipStandardUndeleteInfo(m *Method) *AIPStandardUndeleteInfo {
	if !m.IsSimple && m.OperationInfo == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "undelete")
	if !found || maybeSingular == "" {
		return nil
	}
	if m.InputType == nil ||
		strings.ToLower(m.InputType.Name) != fmt.Sprintf("undelete%srequest", maybeSingular) {
		return nil
	}

	var resourceByType map[string]*Resource
	if m.Model != nil && m.Model.State != nil {
		resourceByType = m.Model.State.ResourceByType
	}

	resourceField := findBestResourceFieldBySingular(m.InputType, resourceByType, maybeSingular)
	if resourceField == nil {
		return nil
	}

	return &AIPStandardUndeleteInfo{
		ResourceNameRequestField: resourceField,
	}
}

func aipStandardCreateInfo(m *Method) *AIPStandardCreateInfo {
	if (!m.IsSimple && !m.IsLRO) || m.InputType == nil || m.ReturnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "create")
	if !found || maybeSingular == "" {
		return nil
	}

	if strings.ToLower(m.InputType.Name) != fmt.Sprintf("create%srequest", maybeSingular) {
		return nil
	}

	if outputResource.Singular != "" &&
		strings.ToLower(outputResource.Singular) != maybeSingular {
		return nil
	}

	parentField := findBestParentFieldByType(m.InputType, outputResource.Type)
	if parentField == nil {
		return nil
	}

	var targetTypeID string
	if outputResource.Self != nil {
		targetTypeID = outputResource.Self.ID
	}
	resourceField := findBodyField(m.InputType, m.PathInfo, targetTypeID, maybeSingular)
	if resourceField == nil {
		return nil
	}

	resourceIDField := findResourceIDField(m.InputType, maybeSingular)

	return &AIPStandardCreateInfo{
		ParentRequestField:     parentField,
		ResourceIDRequestField: resourceIDField,
		ResourceRequestField:   resourceField,
	}
}

func aipStandardUpdateInfo(m *Method) *AIPStandardUpdateInfo {
	if (!m.IsSimple && !m.IsLRO) || m.InputType == nil || m.ReturnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.Name), "update")
	if !found || maybeSingular == "" {
		return nil
	}
	if strings.ToLower(m.InputType.Name) != fmt.Sprintf("update%srequest", maybeSingular) {
		return nil
	}
	if outputResource.Singular != "" &&
		strings.ToLower(outputResource.Singular) != maybeSingular {
		return nil
	}

	var targetTypeID string
	if outputResource.Self != nil {
		targetTypeID = outputResource.Self.ID
	}
	resourceField := findBodyField(m.InputType, m.PathInfo, targetTypeID, maybeSingular)
	if resourceField == nil {
		return nil
	}
	var updateMaskField *Field
	for _, f := range m.InputType.Fields {
		if f.Name == StandardFieldNameForUpdateMask && f.TypezID == ".google.protobuf.FieldMask" {
			updateMaskField = f
			break
		}
	}

	return &AIPStandardUpdateInfo{
		ResourceRequestField:   resourceField,
		UpdateMaskRequestField: updateMaskField,
	}
}

func aipStandardListInfo(m *Method) *AIPStandardListInfo {
	if !m.IsList || m.InputType == nil {
		return nil
	}

	maybePlural, found := strings.CutPrefix(strings.ToLower(m.Name), "list")
	if !found || maybePlural == "" {
		return nil
	}

	if strings.ToLower(m.InputType.Name) != fmt.Sprintf("list%srequest", maybePlural) {
		return nil
	}

	if strings.ToLower(m.OutputType.Name) != fmt.Sprintf("list%sresponse", maybePlural) {
		return nil
	}

	pageableItem := m.OutputType.Pagination.PageableItem
	if pageableItem == nil || pageableItem.MessageType == nil || pageableItem.MessageType.Resource == nil {
		return nil
	}
	resourceType := pageableItem.MessageType.Resource.Type

	parentField := findBestParentFieldByType(m.InputType, resourceType)

	if parentField == nil {
		return nil
	}

	return &AIPStandardListInfo{
		ParentRequestField: parentField,
	}
}

func findBestResourceFieldByType(message *Message, resourcesByType map[string]*Resource, targetType string) *Field {
	var bestField *Field
	for _, field := range message.Fields {
		if field.ResourceReference == nil {
			continue
		}
		if field.ResourceReference.Type == GenericResourceType && field.Name == StandardFieldNameForResourceRef {
			return field
		}
		resource, ok := resourcesByType[field.ResourceReference.Type]
		if !ok {
			continue
		}
		if resource.Type == targetType {
			if field.Name == StandardFieldNameForResourceRef {
				return field
			}
			bestField = field
		}
	}
	return bestField
}

func findBestResourceFieldBySingular(message *Message, resourcesByType map[string]*Resource, targetSingular string) *Field {
	var bestField *Field
	for _, field := range message.Fields {
		if field.ResourceReference == nil {
			continue
		}
		if field.ResourceReference.Type == GenericResourceType && field.Name == StandardFieldNameForResourceRef {
			return field
		}
		resource, ok := resourcesByType[field.ResourceReference.Type]
		if !ok {
			continue
		}
		actualSingular := strings.ToLower(resource.Singular)
		matchesTarget := actualSingular == targetSingular
		if field.Name == StandardFieldNameForResourceRef && (matchesTarget || actualSingular == "") {
			return field
		}
		if matchesTarget {
			bestField = field
		}
	}
	return bestField
}

func findBestParentFieldByType(message *Message, childType string) *Field {
	var bestField *Field
	for _, field := range message.Fields {
		if field.Name == StandardFieldNameForParentResourceRef {
			return field
		}
		if field.ResourceReference != nil && field.ResourceReference.ChildType == childType {
			bestField = field
		}
	}
	return bestField
}

func findBodyField(message *Message, pathInfo *PathInfo, targetTypeID string, singular string) *Field {
	var resourceField *Field
	bodyFieldPath := ""
	if pathInfo != nil {
		bodyFieldPath = pathInfo.BodyFieldPath
	}

	for _, f := range message.Fields {
		if f.Name == bodyFieldPath {
			return f
		}
		if f.Name == singular && f.TypezID == targetTypeID {
			if resourceField == nil {
				resourceField = f
			}
		}
	}
	return resourceField
}

func findResourceIDField(message *Message, singular string) *Field {
	expectedIDName := fmt.Sprintf("%s_id", singular)
	for _, f := range message.Fields {
		if f.Name == expectedIDName && f.Typez == STRING_TYPE {
			return f
		}
	}
	return nil
}

func standardMethodOutputResource(m *Method) *Resource {
	if m.OutputType != nil && m.OutputType.Resource != nil {
		return m.OutputType.Resource
	}
	if m.OperationInfo != nil {
		if lroResponse := m.LongRunningResponseType; lroResponse != nil {
			return lroResponse.Resource
		}
	}
	return nil
}
