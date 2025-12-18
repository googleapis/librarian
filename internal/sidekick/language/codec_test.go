// Copyright 2024 Google LLC
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

package language

import (
	"strings"
	"testing"

	"cloud.google.com/go/librarian/internal/sidekick/api"
	"cloud.google.com/go/librarian/internal/sidekick/sample"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestQueryParams(t *testing.T) {
	field1 := &api.Field{
		Name: "field1",
	}
	field2 := &api.Field{
		Name: "field2",
	}
	request := &api.Message{
		Name: "TestRequest",
		ID:   "..TestRequest",
		Fields: []*api.Field{
			field1, field2,
			{
				Name: "used_in_path",
			},
			{
				Name: "used_in_body",
			},
		},
	}
	binding := &api.PathBinding{
		Verb: "GET",
		QueryParameters: map[string]bool{
			"field1": true,
			"field2": true,
		},
	}
	method := &api.Method{
		Name:      "Test",
		ID:        "..TestService.Test",
		InputType: request,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{binding},
		},
	}

	got := QueryParams(method, binding)
	want := []*api.Field{field1, field2}
	less := func(a, b *api.Field) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatched query parameters (-want, +got):\n%s", diff)
	}
}

func TestPathParams(t *testing.T) {
	test := api.NewTestAPI(
		[]*api.Message{sample.Secret(), sample.UpdateRequest(), sample.CreateRequest()},
		[]*api.Enum{},
		[]*api.Service{sample.Service()},
	)

	less := func(a, b *api.Field) bool { return a.Name < b.Name }

	got := PathParams(sample.MethodCreate(), test.State)
	want := []*api.Field{sample.CreateRequest().Fields[0]}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatched query parameters (-want, +got):\n%s", diff)
	}

	got = PathParams(sample.MethodUpdate(), test.State)
	want = []*api.Field{sample.UpdateRequest().Fields[0]}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatched query parameters (-want, +got):\n%s", diff)
	}
}

func TestFilterSlice(t *testing.T) {
	got := FilterSlice([]string{"a.1", "b.1", "a.2", "b.2"}, func(s string) bool { return strings.HasPrefix(s, "a.") })
	want := []string{"a.1", "a.2"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatched FilterSlice result (-want, +got):\n%s", diff)
	}
}

func TestMapSlice(t *testing.T) {
	got := MapSlice([]string{"a", "aa", "aaa"}, func(s string) int { return len(s) })
	want := []int{1, 2, 3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatched FilterSlice result (-want, +got):\n%s", diff)
	}
}

func TestHasNestedTypes(t *testing.T) {
	for _, test := range []struct {
		input *api.Message
		want  bool
	}{
		{
			input: &api.Message{
				Name: "NoNested",
			},
			want: false,
		},
		{
			input: &api.Message{
				Name:  "WithEnums",
				Enums: []*api.Enum{{Name: "Enum"}},
			},
			want: true,
		},
		{
			input: &api.Message{
				Name:   "WithOneOf",
				OneOfs: []*api.OneOf{{Name: "OneOf"}},
			},
			want: true,
		},
		{
			input: &api.Message{
				Name:     "WithChildMessage",
				Messages: []*api.Message{{Name: "Child"}},
			},
			want: true,
		},
		{
			input: &api.Message{
				Name:     "WithMap",
				Messages: []*api.Message{{Name: "Map", IsMap: true}},
			},
			want: false,
		},
	} {
		got := HasNestedTypes(test.input)
		if got != test.want {
			t.Errorf("mismatched result for HasNestedTypes on %v", test.input)
		}
	}
}

func TestFieldIsMap(t *testing.T) {
	field0 := &api.Field{
		Repeated: false,
		Optional: false,
		Name:     "children",
		ID:       ".test.ParentMessage.children",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.ParentMessage.SingularMapEntry",
	}
	field1 := &api.Field{
		Name:  "singular",
		ID:    ".test.ParentMessage.singular",
		Typez: api.INT32_TYPE,
	}
	field2 := &api.Field{
		Name:    "singular",
		ID:      ".test.ParentMessage.singular",
		Typez:   api.MESSAGE_TYPE,
		TypezID: "invalid",
	}
	parent := &api.Message{
		Name:   "ParentMessage",
		ID:     ".test.ParentMessage",
		Fields: []*api.Field{field0, field1, field2},
	}

	key := &api.Field{
		Name:     "key",
		JSONName: "key",
		ID:       ".test.ParentMessage.SingularMapEntry.key",
		Typez:    api.STRING_TYPE,
	}
	value := &api.Field{
		Name:     "value",
		JSONName: "value",
		ID:       ".test.ParentMessage.SingularMapEntry.value",
		Typez:    api.MESSAGE_TYPE,
		TypezID:  ".test.ParentMessage",
	}
	map_message := &api.Message{
		Name:    "SingularMapEntry",
		Package: "test",
		ID:      ".test.ParentMessage.SingularMapEntry",
		IsMap:   true,
		Fields:  []*api.Field{key, value},
	}
	model := api.NewTestAPI([]*api.Message{parent, map_message}, []*api.Enum{}, []*api.Service{})

	if !FieldIsMap(field0, model.State) {
		t.Errorf("expected FieldIsMap(field0) to be true")
	}
	if FieldIsMap(field1, model.State) {
		t.Errorf("expected FieldIsMap(field1) to be false")
	}
	if FieldIsMap(field2, model.State) {
		t.Errorf("expected FieldIsMap(field2) to be false")
	}
}
