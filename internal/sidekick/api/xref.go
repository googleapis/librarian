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
	for m := range model.AllMaps() {
		if m.key != nil {
			if err := crossReferenceField(model, m.key); err != nil {
				return err
			}
		}
		if m.value != nil {
			if err := crossReferenceField(model, m.value); err != nil {
				return err
			}
		}
	}
	for m := range model.AllMessages() {
		for _, f := range m.fields {
			f.parent = m
			if err := crossReferenceField(model, f); err != nil {
				return err
			}
		}
		for _, o := range m.oneOfs {
			for _, f := range o.fields {
				f.group = o
				f.parent = m
			}
		}
	}
	for m := range model.AllMethods() {
		input := model.Message(m.inputTypeID)
		if input == nil {
			return fmt.Errorf("cannot find input type %s for method %s", m.inputTypeID, m.id)
		}
		output := model.Message(m.outputTypeID)
		if output == nil {
			return fmt.Errorf("cannot find output type %s for method %s", m.outputTypeID, m.id)
		}
		m.inputType = input
		m.outputType = output
		if m.operationInfo != nil {
			m.operationInfo.method = m
		}
		for _, signature := range m.signatures {
			signature.method = m
			for _, name := range signature.names {
				idx := slices.IndexFunc(input.fields, func(f *Field) bool { return f.name == name })
				if idx == -1 {
					return fmt.Errorf("cannot find field %s in method signature for method %s", name, m.id)
				}
				signature.fields = append(signature.fields, input.fields[idx])
			}
		}
	}
	for s := range model.AllServices() {
		s.model = model
		for _, m := range s.methods {
			m.model = model
			m.service = s
			if source := model.Service(m.sourceServiceID); source != nil {
				m.sourceService = source
			} else {
				// Default to the regular service. OpenAPI does not define the
				// services for mixins.
				m.sourceService = s
			}
		}
	}
	enrichSamples(model)
	return nil
}

// enrichSamples populates the API model with information useful for generating code samples.
// This includes selecting representative enum values and optimal fields for oneof structures.
func enrichSamples(model *API) {
	for e := range model.AllEnums() {
		enrichEnumSamples(e)
	}

	for m := range model.AllMessages() {
		for _, o := range m.oneOfs {
			if len(o.fields) > 0 {
				o.exampleField = slices.MaxFunc(o.fields, sortOneOfFieldForExamples)
			}
		}
		for _, f := range m.fields {
			enrichWithResourceNamePattern(f, m, model)
		}
	}

	for m := range model.AllMethods() {
		enrichMethodSamples(m)
	}

	for _, s := range model.Services {
		s.quickstartMethod = findQuickstartMethod(s)
	}
	model.QuickstartService = findQuickstartService(model)
}

func findQuickstartMethod(s *Service) *Method {
	// Priority: List > Get > Create > Delete > Update
	priorities := []func(m *Method) bool{
		func(m *Method) bool { return m.isAIPStandardList },
		func(m *Method) bool { return m.isAIPStandardGet },
		func(m *Method) bool { return m.isAIPStandardCreate },
		func(m *Method) bool { return m.isAIPStandardDelete },
		func(m *Method) bool { return m.isAIPStandardUpdate },
		// Fallback for when no standard AIP method is available: any method that is not streaming.
		func(m *Method) bool { return !m.clientSideStreaming && !m.serverSideStreaming },
		// Fallback for streaming methods when no unary methods exist.
		func(m *Method) bool { return !m.clientSideStreaming && m.serverSideStreaming },
		func(m *Method) bool { return m.clientSideStreaming && m.serverSideStreaming },
	}

	strippedServiceName := strings.TrimSuffix(s.name, "Service")
	lowerStripped := strings.ToLower(strippedServiceName)

	for _, isType := range priorities {
		var nonDeprecated []*Method
		var deprecated []*Method

		for _, m := range s.methods {
			if isType(m) {
				if m.deprecated {
					deprecated = append(deprecated, m)
				} else {
					nonDeprecated = append(nonDeprecated, m)
				}
			}
		}

		searchList := nonDeprecated
		if len(searchList) == 0 {
			searchList = deprecated
		}

		if len(searchList) == 0 {
			continue
		}

		if len(searchList) == 1 {
			return searchList[0]
		}

		// Tie-breaking: Substring match on method name
		for _, m := range searchList {
			if strings.Contains(strings.ToLower(m.name), lowerStripped) {
				return m
			}
		}

		// Tie-breaking: Resource singular/plural match
		for _, m := range searchList {
			res := standardMethodOutputResource(m)
			if res != nil {
				if strings.ToLower(res.Singular()) == lowerStripped || strings.ToLower(res.Plural()) == lowerStripped {
					return m
				}
			}
		}

		// Default to first candidate if no tie-breaker matches
		return searchList[0]
	}
	return nil
}

func findQuickstartService(api *API) *Service {
	if len(api.Services) == 0 {
		return nil
	}

	var nonDeprecated []*Service
	var deprecated []*Service

	for _, s := range api.Services {
		if len(s.methods) > 0 {
			if s.deprecated {
				deprecated = append(deprecated, s)
			} else {
				nonDeprecated = append(nonDeprecated, s)
			}
		}
	}

	searchList := nonDeprecated
	if len(searchList) == 0 {
		searchList = deprecated
	}

	if len(searchList) == 0 {
		return api.Services[0]
	}

	if len(searchList) == 1 {
		return searchList[0]
	}

	// Prefer services with a QuickstartMethod that is an AIP standard method
	var servicesWithStandardQuickstart []*Service
	// Fallback to services with ANY QuickstartMethod
	var servicesWithAnyQuickstart []*Service

	for _, s := range searchList {
		if s.quickstartMethod != nil {
			servicesWithAnyQuickstart = append(servicesWithAnyQuickstart, s)
			if s.quickstartMethod.isAIPStandardList ||
				s.quickstartMethod.isAIPStandardGet ||
				s.quickstartMethod.isAIPStandardCreate ||
				s.quickstartMethod.isAIPStandardDelete ||
				s.quickstartMethod.isAIPStandardUpdate {
				servicesWithStandardQuickstart = append(servicesWithStandardQuickstart, s)
			}
		}
	}

	if len(servicesWithStandardQuickstart) > 0 {
		searchList = servicesWithStandardQuickstart
	} else if len(servicesWithAnyQuickstart) > 0 {
		searchList = servicesWithAnyQuickstart
	}

	lowerApiName := strings.ToLower(api.Name)
	for _, s := range searchList {
		if strings.Contains(strings.ToLower(s.name), lowerApiName) {
			return s
		}
	}

	return searchList[0]
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
		return !v.Deprecated() && v.Number() != 0
	}

	for _, ev := range e.values {
		// A simple heuristic to avoid duplicates.
		// This is not perfect, but it should handle the most common cases.
		name := strcase.ToCamel(strings.ToLower(ev.Name()))
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

	e.valuesForExamples = make([]*SampleValue, len(result))
	for i, ev := range result {
		e.valuesForExamples[i] = NewSampleValue(ev, i)
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
	if v := compare(f1.deprecated, f2.deprecated); v != 0 {
		return v
	}
	if v := compare(f1.mapz, f2.mapz); v != 0 {
		return v
	}
	if v := compare(f1.repeated, f2.repeated); v != 0 {
		return v
	}
	return compare(f1.messageType != nil, f2.messageType != nil)
}

func enrichMethodSamples(m *Method) {
	// Methods with AIP-151 LRO annotations *OR* discovery LRO annotations are LROs.
	m.isLRO = m.operationInfo != nil || m.discoveryLro != nil
	m.isStreaming = m.clientSideStreaming || m.serverSideStreaming
	// A simple method is not paginated, not streaming and not an LRO.
	m.isSimple = m.pagination == nil && !m.isStreaming && !m.isLRO

	if m.sourceServiceID == ".google.longrunning.Operations" &&
		m.name == "GetOperation" &&
		m.service != nil && m.service.packagez != "google.longrunning" {
		m.isLroPoller = true
	}

	if m.operationInfo != nil && m.model != nil {
		m.longRunningResponseType = m.model.Message(m.operationInfo.ResponseTypeID())
	}

	m.longRunningReturnsEmpty = m.longRunningResponseType != nil && m.longRunningResponseType.id == ".google.protobuf.Empty"

	m.isList = m.outputType != nil && m.outputType.pagination != nil

	if m.sampleInfo = aipStandardGetInfo(m); m.sampleInfo != nil {
		m.isAIPStandardGet = true
	} else if m.sampleInfo = aipStandardDeleteInfo(m); m.sampleInfo != nil {
		m.isAIPStandardDelete = true
	} else if m.sampleInfo = aipStandardUndeleteInfo(m); m.sampleInfo != nil {
		m.isAIPStandardUndelete = true
	} else if m.sampleInfo = aipStandardCreateInfo(m); m.sampleInfo != nil {
		m.isAIPStandardCreate = true
	} else if m.sampleInfo = aipStandardUpdateInfo(m); m.sampleInfo != nil {
		m.isAIPStandardUpdate = true
	} else if m.sampleInfo = aipStandardListInfo(m); m.sampleInfo != nil {
		m.isAIPStandardList = true
	}

	m.isAIPStandard = m.sampleInfo != nil
}

func enrichWithResourceNamePattern(f *Field, m *Message, model *API) {
	var resource *Resource
	truncateChild := false

	if f.resourceReference != nil {
		// The field is a resource reference.
		if res := model.Resource(f.resourceReference.Type()); res != nil {
			resource = res
		} else if res := model.Resource(f.resourceReference.ChildType()); res != nil {
			resource = res
			truncateChild = true
		}
	} else if f.name == StandardFieldNameForResourceRef {
		// The field is the `name` field.
		resource = m.resource
	}

	if resource != nil && len(resource.Patterns()) > 0 {
		f.resourceNamePattern = toResourceNamePattern(resource.Patterns()[0], truncateChild)
	}
}

// toResourceNamePattern converts a ResourcePattern into a ResourceNamePattern.
func toResourceNamePattern(pattern ResourcePattern, skipLast bool) *ResourceNamePattern {
	var segments []ResourceNameSegment
	for i := 0; i < len(pattern); i++ {
		s := pattern[i]
		if strings.HasPrefix(s.Literal(), "//") {
			continue
		}
		seg := ResourceNameSegment{
			literal: s.Literal(),
		}
		if s.Variable() != nil && len(s.Variable().FieldPath()) > 0 {
			// The parser used for parsing resource name patterns is the same used for
			// parsing HTTP rules. Resource patterns do not have "field paths",
			// instead they only have "placeholders" for each of the resource IDs of
			// the resources that are part of the resource name hierarchy.
			// These placeholders end up on the first, and only, field path
			// that results when parsing a segment of a resource name pattern.
			seg.variable = s.Variable().FieldPath()[0]
		}

		// Try to combine with next segment if this is a literal and next is variable
		if s.Literal() != "" && i+1 < len(pattern) && pattern[i+1].Variable() != nil {
			if len(pattern[i+1].Variable().FieldPath()) > 0 {
				seg.variable = pattern[i+1].Variable().FieldPath()[0]
			}
			i++
		}

		// Clean up leading and trailing slashes
		seg.literal = strings.TrimSuffix(strings.TrimPrefix(seg.literal, "/"), "/")

		segments = append(segments, seg)
	}
	if skipLast && len(segments) > 0 {
		segments = segments[:len(segments)-1]
	}
	return NewResourceNamePattern(segments...)
}

func aipStandardGetInfo(m *Method) *SampleInfo {
	if !m.isSimple || m.inputType == nil || m.returnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.name), "get")
	if !found || maybeSingular == "" {
		return nil
	}
	if strings.ToLower(m.inputType.name) != fmt.Sprintf("get%srequest", maybeSingular) {
		return nil
	}

	if outputResource.Singular() != "" &&
		strings.ToLower(outputResource.Singular()) != maybeSingular {
		return nil
	}

	resourceField := findBestResourceFieldByType(m.inputType, m.model, outputResource.Type())

	if resourceField == nil {
		return nil
	}

	return &SampleInfo{
		resourceNameField:     resourceField,
		isRequestResourceName: true,
	}
}

func aipStandardDeleteInfo(m *Method) *SampleInfo {
	if !m.isSimple && m.operationInfo == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.name), "delete")
	if !found || maybeSingular == "" {
		return nil
	}
	if m.inputType == nil ||
		strings.ToLower(m.inputType.name) != fmt.Sprintf("delete%srequest", maybeSingular) {
		return nil
	}

	resourceField := findBestResourceFieldBySingular(m.inputType, m.model, maybeSingular)
	if resourceField == nil {
		return nil
	}

	return &SampleInfo{
		resourceNameField:     resourceField,
		isRequestResourceName: true,
	}
}

func aipStandardUndeleteInfo(m *Method) *SampleInfo {
	if !m.isSimple && m.operationInfo == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.name), "undelete")
	if !found || maybeSingular == "" {
		return nil
	}
	if m.inputType == nil ||
		strings.ToLower(m.inputType.name) != fmt.Sprintf("undelete%srequest", maybeSingular) {
		return nil
	}

	resourceField := findBestResourceFieldBySingular(m.inputType, m.model, maybeSingular)
	if resourceField == nil {
		return nil
	}

	return &SampleInfo{
		resourceNameField:     resourceField,
		isRequestResourceName: true,
	}
}

func aipStandardCreateInfo(m *Method) *SampleInfo {
	if (!m.isSimple && !m.isLRO) || m.inputType == nil || m.returnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.name), "create")
	if !found || maybeSingular == "" {
		return nil
	}

	if strings.ToLower(m.inputType.name) != fmt.Sprintf("create%srequest", maybeSingular) {
		return nil
	}

	if outputResource.Singular() != "" &&
		strings.ToLower(outputResource.Singular()) != maybeSingular {
		return nil
	}

	parentField := findBestParentFieldByType(m.inputType, outputResource.Type())
	if parentField == nil {
		return nil
	}

	var targetTypeID string
	if outputResource.Self() != nil {
		targetTypeID = outputResource.Self().id
	}
	resourceField := findBodyField(m.inputType, m.pathInfo, targetTypeID, maybeSingular)
	if resourceField == nil {
		return nil
	}

	resourceIDField := findResourceIDField(m.inputType, maybeSingular)

	info := &SampleInfo{
		resourceNameField:     parentField,
		isRequestResourceName: true,
		messageField:          resourceField,
	}
	if resourceIDField != nil {
		info.resourceIDField = resourceIDField
	}
	return info
}

func aipStandardUpdateInfo(m *Method) *SampleInfo {
	if (!m.isSimple && !m.isLRO) || m.inputType == nil || m.returnsEmpty {
		return nil
	}
	outputResource := standardMethodOutputResource(m)
	if outputResource == nil {
		return nil
	}

	maybeSingular, found := strings.CutPrefix(strings.ToLower(m.name), "update")
	if !found || maybeSingular == "" {
		return nil
	}
	if strings.ToLower(m.inputType.name) != fmt.Sprintf("update%srequest", maybeSingular) {
		return nil
	}
	if outputResource.Singular() != "" &&
		strings.ToLower(outputResource.Singular()) != maybeSingular {
		return nil
	}

	var targetTypeID string
	if outputResource.Self() != nil {
		targetTypeID = outputResource.Self().id
	}
	resourceField := findBodyField(m.inputType, m.pathInfo, targetTypeID, maybeSingular)
	if resourceField == nil {
		return nil
	}
	var updateMaskField *Field
	for _, f := range m.inputType.fields {
		if f.name == StandardFieldNameForUpdateMask && f.typezID == ".google.protobuf.FieldMask" {
			updateMaskField = f
			break
		}
	}

	var resourceNameField *Field
	if resourceField.messageType != nil {
		for _, f := range resourceField.messageType.fields {
			if f.name == StandardFieldNameForResourceRef {
				resourceNameField = f
				break
			}
		}
	}

	return &SampleInfo{
		resourceNameField:     resourceNameField,
		isMessageResourceName: true,
		messageField:          resourceField,
		updateMaskField:       updateMaskField,
	}
}

func aipStandardListInfo(m *Method) *SampleInfo {
	if !m.isList || m.inputType == nil {
		return nil
	}

	maybePlural, found := strings.CutPrefix(strings.ToLower(m.name), "list")
	if !found || maybePlural == "" {
		return nil
	}

	if strings.ToLower(m.inputType.name) != fmt.Sprintf("list%srequest", maybePlural) {
		return nil
	}

	if strings.ToLower(m.outputType.name) != fmt.Sprintf("list%sresponse", maybePlural) {
		return nil
	}

	pageableItem := m.outputType.pagination.pageableItem
	if pageableItem == nil || pageableItem.messageType == nil || pageableItem.messageType.resource == nil {
		return nil
	}
	resourceType := pageableItem.messageType.resource.Type()

	parentField := findBestParentFieldByType(m.inputType, resourceType)

	if parentField == nil {
		return nil
	}

	return &SampleInfo{
		resourceNameField:     parentField,
		isRequestResourceName: true,
	}
}

func findBestResourceFieldByType(message *Message, model *API, targetType string) *Field {
	var bestField *Field
	for _, field := range message.fields {
		if field.resourceReference == nil {
			continue
		}
		if field.resourceReference.Type() == GenericResourceType && field.name == StandardFieldNameForResourceRef {
			return field
		}
		resource := model.Resource(field.resourceReference.Type())
		if resource == nil {
			continue
		}
		if resource.Type() == targetType {
			if field.name == StandardFieldNameForResourceRef {
				return field
			}
			bestField = field
		}
	}
	return bestField
}

func findBestResourceFieldBySingular(message *Message, model *API, targetSingular string) *Field {
	var bestField *Field
	for _, field := range message.fields {
		if field.resourceReference == nil {
			continue
		}
		if field.resourceReference.Type() == GenericResourceType && field.name == StandardFieldNameForResourceRef {
			return field
		}
		resource := model.Resource(field.resourceReference.Type())
		if resource == nil {
			continue
		}
		actualSingular := strings.ToLower(resource.Singular())
		matchesTarget := actualSingular == targetSingular
		if field.name == StandardFieldNameForResourceRef && (matchesTarget || actualSingular == "") {
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
	for _, field := range message.fields {
		if field.name == StandardFieldNameForParentResourceRef {
			return field
		}
		if field.resourceReference != nil && field.resourceReference.ChildType() == childType {
			bestField = field
		}
	}
	return bestField
}

func findBodyField(message *Message, pathInfo *PathInfo, targetTypeID string, singular string) *Field {
	var resourceField *Field
	bodyFieldPath := ""
	if pathInfo != nil {
		bodyFieldPath = pathInfo.BodyFieldPath()
	}

	for _, f := range message.fields {
		if f.name == bodyFieldPath {
			return f
		}
		if f.name == singular && f.typezID == targetTypeID {
			if resourceField == nil {
				resourceField = f
			}
		}
	}
	return resourceField
}

func findResourceIDField(message *Message, singular string) *Field {
	expectedIDName := fmt.Sprintf("%s_id", singular)
	for _, f := range message.fields {
		if f.name == expectedIDName && f.typez == TypezString {
			return f
		}
	}
	return nil
}

func standardMethodOutputResource(m *Method) *Resource {
	if m.outputType != nil && m.outputType.resource != nil {
		return m.outputType.resource
	}
	if m.operationInfo != nil {
		if lroResponse := m.longRunningResponseType; lroResponse != nil {
			return lroResponse.resource
		}
	}
	return nil
}

func crossReferenceField(model *API, f *Field) error {
	switch f.typez {
	case TypezMessage:
		t := model.Message(f.typezID)
		if t == nil {
			return fmt.Errorf("unknown message type %s for field %s", f.typezID, f.id)
		}
		f.messageType = t
	case TypezEnum:
		t := model.Enum(f.typezID)
		if t == nil {
			return fmt.Errorf("unknown enum type %s for field %s", f.typezID, f.id)
		}
		f.enumType = t
	case TypezMap:
		t := model.Map(f.typezID)
		if t == nil {
			return fmt.Errorf("unknown map type %s for field %s", f.typezID, f.id)
		}
		f.mapType = t
	}
	return nil
}
