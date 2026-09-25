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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestScopesService(t *testing.T) {
	service := NewTestService("Service")
	got := service.Scopes()
	want := []string{"test.Service", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesMessage(t *testing.T) {
	child := NewTestMessage("Child")
	parent := NewTestMessage("Parent").WithMessages(child)

	got := parent.Scopes()
	want := []string{"test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = child.Scopes()
	want = []string{"test.Parent.Child", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesEnum(t *testing.T) {
	enum := NewTestEnum("Enum")

	got := enum.Scopes()
	want := []string{"test.Enum", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesEnumInMessage(t *testing.T) {
	child := NewTestEnum("Child")
	_ = NewTestMessage("Parent").WithEnums(child)

	got := child.Scopes()
	want := []string{"test.Parent.Child", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesEnumValue(t *testing.T) {
	enumValue := NewTestEnumValue("EV", 0)
	_ = NewTestEnum("Enum").WithValues(enumValue)

	got := enumValue.Scopes()
	want := []string{"test.Enum", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesEnumValueInMessage(t *testing.T) {
	enumValue := NewTestEnumValue("EV", 0)
	enum := NewTestEnum("Enum").WithValues(enumValue)
	_ = NewTestMessage("Parent").WithEnums(enum)

	got := enumValue.Scopes()
	want := []string{"test.Parent.Enum", "test.Parent", "test"}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestScopesField(t *testing.T) {
	standardField := NewTestField("field")
	_ = NewTestMessage("Parent").WithFields(standardField)

	nilParentField := NewTestField("field")
	nilParentField.ID = ".test.Parent.field"

	for _, test := range []struct {
		name  string
		field *Field
		want  []string
	}{
		{
			name:  "standard",
			field: standardField,
			want:  []string{"test.Parent", "test"},
		},
		{
			name:  "nil parent",
			field: nilParentField,
			want:  []string{"test.Parent", "test"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.field.Scopes()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestScopesMethod(t *testing.T) {
	standardMethod := NewTestMethod("Method")
	_ = NewTestService("Service").WithMethods(standardMethod)
	for _, test := range []struct {
		name   string
		method *Method
		want   []string
	}{
		{
			name:   "standard",
			method: standardMethod,
			want:   []string{"test.Service", "test"},
		},
		{
			name:   "none set",
			method: NewTestMethod("Method").WithID(".test2.Service2.Method"),
			want:   []string{"test2.Service2", "test2"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.method.Scopes()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestScopesOneOf(t *testing.T) {
	field := NewTestField("field")
	withFieldsOneOf := NewTestOneOf("oneof").WithFields(field)
	emptyOneOf := NewTestOneOf("empty_oneof")
	_ = NewTestMessage("Parent").WithOneOfs(withFieldsOneOf, emptyOneOf)

	for _, test := range []struct {
		name  string
		oneof *OneOf
		want  []string
	}{
		{
			name:  "with fields",
			oneof: withFieldsOneOf,
			want:  []string{"test.Parent", "test"},
		},
		{
			name:  "empty",
			oneof: emptyOneOf,
			want:  []string{"test.Parent", "test"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.oneof.Scopes()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFallbackScopes(t *testing.T) {
	for _, test := range []struct {
		name string
		id   string
		want []string
	}{
		{
			name: "empty",
			id:   "",
			want: []string{},
		},
		{
			name: "no dot",
			id:   "foo",
			want: []string{},
		},
		{
			name: "single dot",
			id:   ".foo",
			want: []string{},
		},
		{
			name: "two parts with dot",
			id:   ".foo.bar",
			want: []string{"foo"},
		},
		{
			name: "three parts with dot",
			id:   ".foo.bar.baz",
			want: []string{"foo.bar", "foo"},
		},
		{
			name: "two parts no dot",
			id:   "foo.bar",
			want: []string{"foo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := fallbackScopes(test.id)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
