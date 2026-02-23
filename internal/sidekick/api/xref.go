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
