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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEnumValue(t *testing.T) {
	parent := NewEnumBuilder().WithName("State").Build()
	got := NewEnumValueBuilder().
		WithName("ACTIVE").
		WithID(".test.State.ACTIVE").
		WithDocumentation("Active state.").
		WithNumber(1).
		WithDeprecated(true).
		WithParent(parent).
		Build()
	got.SetAnnotations("ev-codec")

	want := &EnumValue{
		name:          "ACTIVE",
		id:            ".test.State.ACTIVE",
		documentation: "Active state.",
		number:        1,
		deprecated:    true,
		parent:        parent,
		codec:         "ev-codec",
	}
	opts := cmp.Options{
		cmp.AllowUnexported(Enum{}, EnumValue{}, SampleValue{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return len(p) > 2 && p.Last().String() == ".parent"
		}, cmp.Ignore()),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("EnumValue mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Documentation() != want.documentation || got.Number() != want.number || !got.Deprecated() || got.Parent() != parent || got.Codec() != want.codec {
		t.Errorf("unexpected EnumValue getters: %+v", got)
	}

	gotSV := NewSampleValue(got, 3)
	wantSV := &SampleValue{
		enumValue: got,
		index:     3,
	}
	if diff := cmp.Diff(wantSV, gotSV, opts); diff != "" {
		t.Errorf("SampleValue mismatch (-want +got):\n%s", diff)
	}
	if gotSV.EnumValue() != got || gotSV.Index() != 3 {
		t.Errorf("unexpected SampleValue getters: %+v", gotSV)
	}
}

func TestEnum(t *testing.T) {
	ev := NewEnumValueBuilder().WithName("ACTIVE").WithNumber(1).Build()
	sv := NewSampleValue(ev, 0)
	parent := NewMessageBuilder().WithName("Parent").Build()

	got := NewEnumBuilder().
		WithName("State").
		WithID(".test.State").
		WithPackage("test").
		WithDocumentation("State enum.").
		WithDeprecated(true).
		WithValues(ev).
		WithUniqueNumberValues(ev).
		WithValuesForExamples(sv).
		WithParent(parent).
		Build()

	got.SetAnnotations("enum-codec")

	want := &Enum{
		name:               "State",
		id:                 ".test.State",
		packagez:           "test",
		documentation:      "State enum.",
		deprecated:         true,
		values:             []*EnumValue{ev},
		uniqueNumberValues: []*EnumValue{ev},
		valuesForExamples:  []*SampleValue{sv},
		parent:             parent,
		codec:              "enum-codec",
	}
	opts := cmp.Options{
		cmp.AllowUnexported(Enum{}, EnumValue{}, SampleValue{}, Message{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return len(p) > 2 && p.Last().String() == ".parent"
		}, cmp.Ignore()),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Errorf("Enum mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Package() != want.packagez || got.Documentation() != want.documentation || !got.Deprecated() || got.Parent() != parent || got.Codec() != want.codec {
		t.Errorf("unexpected Enum getters: %+v", got)
	}
	if diff := cmp.Diff(want.values, got.Values(), opts); diff != "" {
		t.Errorf("Values() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.uniqueNumberValues, got.UniqueNumberValues(), opts); diff != "" {
		t.Errorf("UniqueNumberValues() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.valuesForExamples, got.ValuesForExamples(), opts); diff != "" {
		t.Errorf("ValuesForExamples() mismatch (-want +got):\n%s", diff)
	}

	got.Values()[0] = nil
	got.UniqueNumberValues()[0] = nil
	got.ValuesForExamples()[0] = nil
	if got.Values()[0] != ev || got.UniqueNumberValues()[0] != ev || got.ValuesForExamples()[0] != sv {
		t.Errorf("Enum slice getters should return defensive copies")
	}
}
