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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestQueryParams(t *testing.T) {
	field1 := api.NewTestField("field1")
	field2 := api.NewTestField("field2")
	request := api.NewTestMessage("TestRequest").
		WithPackage("").
		WithFields(
			field1,
			field2,
			api.NewTestField("used_in_path"),
			api.NewTestField("used_in_body"),
		)
	binding := api.NewTestPathBinding("GET", nil).
		WithQueryParameters(map[string]bool{
			"field1": true,
			"field2": true,
		})
	method := api.NewTestMethod("Test").
		WithID("..TestService.Test").
		WithInput(request).
		WithBindings(binding)

	got := QueryParams(method, binding)
	want := []*api.Field{field1, field2}
	less := func(a, b *api.Field) bool { return a.Name < b.Name }
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPathParams(t *testing.T) {
	test := api.NewTestAPI(
		[]*api.Message{sample.Secret(), sample.UpdateRequest(), sample.CreateRequest()},
		[]*api.Enum{},
		[]*api.Service{sample.Service()},
	)

	less := func(a, b *api.Field) bool { return a.Name < b.Name }

	got, err := PathParams(sample.MethodCreate(), test)
	if err != nil {
		t.Fatal(err)
	}
	want := []*api.Field{sample.CreateRequest().Fields[0]}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got, err = PathParams(sample.MethodUpdate(), test)
	if err != nil {
		t.Fatal(err)
	}
	want = []*api.Field{sample.UpdateRequest().Fields[0]}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterSlice(t *testing.T) {
	got := FilterSlice([]string{"a.1", "b.1", "a.2", "b.2"}, func(s string) bool { return strings.HasPrefix(s, "a.") })
	want := []string{"a.1", "a.2"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMapSlice(t *testing.T) {
	got := MapSlice([]string{"a", "aa", "aaa"}, func(s string) int { return len(s) })
	want := []int{1, 2, 3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestHasNestedTypes(t *testing.T) {
	for _, test := range []struct {
		input *api.Message
		want  bool
	}{
		{
			input: api.NewTestMessage("NoNested"),
			want:  false,
		},
		{
			input: api.NewTestMessage("WithEnums").
				WithEnums(api.NewTestEnum("Enum")),
			want: true,
		},
		{
			input: api.NewTestMessage("WithOneOf").
				WithOneOfs(api.NewTestOneOf("OneOf")),
			want: true,
		},
		{
			input: api.NewTestMessage("WithChildMessage").
				WithMessages(api.NewTestMessage("Child")),
			want: true,
		},
		{
			input: api.NewTestMessage("WithMap").
				WithMessages(api.NewTestMapMessage("Map", api.TypezString, api.TypezString)),
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
	parent := api.NewTestMessage("ParentMessage")
	mapMessage := api.NewTestMapMessageWithFields(
		"SingularMapEntry",
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithMessageType(parent),
	)
	parent.WithMessages(mapMessage)

	field0 := api.NewTestField("children").WithMessageType(mapMessage)
	field1 := api.NewTestField("singular").WithType(api.TypezInt32)
	field2 := api.NewTestField("singular").WithType(api.TypezMessage).WithTypezID("invalid")
	parent.WithFields(field0, field1, field2)

	model := api.NewTestAPI([]*api.Message{parent, mapMessage}, nil, nil)

	if !FieldIsMap(field0, model) {
		t.Errorf("expected FieldIsMap(field0) to be true")
	}
	if FieldIsMap(field1, model) {
		t.Errorf("expected FieldIsMap(field1) to be false")
	}
	if FieldIsMap(field2, model) {
		t.Errorf("expected FieldIsMap(field2) to be false")
	}
}
