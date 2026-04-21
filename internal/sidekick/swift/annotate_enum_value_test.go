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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateEnumValue_WithDocs(t *testing.T) {
	enum := &api.Enum{Name: "Color"}
	ev := &api.EnumValue{Name: "COLOR_RED", Number: 1, Documentation: "Red color", Parent: enum}
	enum.Values = []*api.EnumValue{ev}
	enum.UniqueNumberValues = enum.Values

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})
	if _, err := annotateModel(codec); err != nil {
		t.Fatal(err)
	}

	ann, ok := ev.Codec.(*enumValueAnnotations)
	if !ok {
		t.Fatal("expected enumValueAnnotations")
	}

	want := &enumValueAnnotations{
		CaseName:    "red",
		Number:      1,
		StringValue: "COLOR_RED",
		DocLines:    []string{"Red color"},
	}
	if diff := cmp.Diff(want, ann); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateEnumValue_Multiple(t *testing.T) {
	enum := &api.Enum{Name: "Color"}
	enum.Values = []*api.EnumValue{
		{Name: "COLOR_RED", Number: 1, Parent: enum},
		{Name: "COLOR_GREEN", Number: 2, Parent: enum},
	}
	enum.UniqueNumberValues = enum.Values

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})
	if _, err := annotateModel(codec); err != nil {
		t.Fatal(err)
	}

	var got []*enumValueAnnotations
	for _, ev := range enum.Values {
		got = append(got, ev.Codec.(*enumValueAnnotations))
	}

	want := []*enumValueAnnotations{
		{
			CaseName:    "red",
			Number:      1,
			StringValue: "COLOR_RED",
		},
		{
			CaseName:    "green",
			Number:      2,
			StringValue: "COLOR_GREEN",
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateEnumValue_Aliases(t *testing.T) {
	enum := &api.Enum{Name: "Color"}
	// This may seem weird, but they do happen in Google Cloud APIs, see:
	//     https://github.com/search?q=repo%3Agoogleapis%2Fgoogleapis+%22option+allow_alias+%3D+true%3B%22&type=code
	enum.Values = []*api.EnumValue{
		{Name: "RED_NEW", Number: 1, Parent: enum},
		{Name: "RED_OLD", Number: 1, Parent: enum}, // Alias with same number
	}
	enum.UniqueNumberValues = []*api.EnumValue{enum.Values[0]}

	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{enum}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})
	if _, err := annotateModel(codec); err != nil {
		t.Fatal(err)
	}

	var got []*enumValueAnnotations
	for _, ev := range enum.Values {
		got = append(got, ev.Codec.(*enumValueAnnotations))
	}

	want := []*enumValueAnnotations{
		{
			CaseName:    "redNew",
			Number:      1,
			StringValue: "RED_NEW",
		},
		{
			CaseName:    "redNew",
			Number:      1,
			StringValue: "RED_OLD",
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in Values (-want +got):\n%s", diff)
	}
}
